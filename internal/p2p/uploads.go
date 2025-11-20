package p2p

import (
	"context"
	"fmt"
	"pinshare/internal/psfs"
	"pinshare/internal/store"
	"strings"
)

func ProcessUploads(ctx context.Context, folderPath string) {
	files, err := psfs.ListFiles(folderPath)
	if err != nil {
		return
	}

	// Process each file concurrently with individual cancellation support
	for _, fileName := range files {
		// Check if parent context is cancelled before starting new file
		select {
		case <-ctx.Done():
			fmt.Println("[INFO] Upload processing cancelled by context")
			return
		default:
		}

		// Create individual context for this file upload
		fileCtx, cancel := context.WithCancel(ctx)

		// Start tracking with cancel function
		store.StatusManager.StartUploadWithContext(fileName, cancel)

		// Launch file processing in goroutine for concurrency
		go processFile(fileCtx, folderPath, fileName)
	}
}

// processFile handles the upload processing for a single file
func processFile(ctx context.Context, folderPath, f string) {
	// Check if context is cancelled at start
	select {
	case <-ctx.Done():
		fmt.Println("[INFO] Upload cancelled before starting for file: " + f)
		store.StatusManager.FailUpload(f, "Upload cancelled")
		return
	default:
	}

		ftype, err := psfs.ValidateFileType(folderPath + "/" + f)
		if err != nil {
			fmt.Println("[ERROR] func ValidateFileType() error " + string(err.Error()))
			store.StatusManager.FailUpload(f, "File type validation failed: "+err.Error())
			return
		}
		if ftype {
			fmt.Println("[INFO] File type valid for file: " + f)
			store.StatusManager.UpdateStatus(f, "hashing", 10, "")

			fsha256, err := psfs.GetSHA256(folderPath + "/" + f)
			if err != nil {
				fmt.Println("[ERROR] func GetSha256() error " + string(err.Error()))
				store.StatusManager.FailUpload(f, "Failed to calculate SHA256: "+err.Error())
				return
			}

			var fresult bool
			if appconfInstance.FFIgnoreUploadsInMetadata {

				_, exists := store.GlobalStore.GetFile(fsha256)
				if exists {
					fmt.Printf("[WARNING] File already exists in GlobalStore with SHA256: %s \n", fsha256)
					store.StatusManager.FailUpload(f, "File already exists in GlobalStore")
					return
				} else {

					if appconfInstance.SecurityCapability > 0 {
						fmt.Println("[INFO] File Security checking file: " + f + " with SHA256: " + fsha256)
						store.StatusManager.UpdateStatus(f, "scanning", 30, fsha256)

						// Check for context cancellation before security scan
						select {
						case <-ctx.Done():
							fmt.Println("[INFO] Upload cancelled during security scan preparation for file: " + f)
							store.StatusManager.FailUpload(f, "Upload cancelled")
							return
						default:
						}

						var result bool
						var err error
						// TODO: 				if appconfInstance.SecurityCapability [1 2 3 4]
						if appconfInstance.SecurityCapability <= 3 {
							result, err = psfs.ClamScanFileClean(ctx, folderPath+"/"+f)
							if err != nil {
								// Check if error is due to cancellation
								if ctx.Err() != nil {
									fmt.Println("[INFO] ClamScan cancelled for file: " + f)
									store.StatusManager.FailUpload(f, "Upload cancelled during security scan")
									return
								}
								fmt.Println("[ERROR] (ClamScanFileClean) " + string(err.Error()))
								store.StatusManager.FailUpload(f, "Security scan failed: "+err.Error())
								return
							}
						}

						if appconfInstance.SecurityCapability == 4 {
							if appconfInstance.FFSkipVT {
								result = true
							} else {
								result, err = psfs.GetVirusTotalWSVerdictByHash(ctx, fsha256) // true == safe
								if err != nil {
									// Check if error is due to cancellation
									if ctx.Err() != nil {
										fmt.Println("[INFO] VirusTotal scan cancelled for file: " + f)
										store.StatusManager.FailUpload(f, "Upload cancelled during VirusTotal scan")
										return
									}
									fmt.Println("[ERROR] (GetVirusTotalVerdictByHash) " + string(err.Error()))
									store.StatusManager.FailUpload(f, "VirusTotal scan failed: "+err.Error())
									return
								}
							}
						}

						// fmt.Println("[INFOSEC] File Security check passed for file: " + f + " with SHA256: " + fsha256)
						fresult = result
					}

				}
			}

			if fresult {
				// Check for context cancellation before IPFS upload
				select {
				case <-ctx.Done():
					fmt.Println("[INFO] Upload cancelled before IPFS upload for file: " + f)
					store.StatusManager.FailUpload(f, "Upload cancelled")
					return
				default:
				}

				store.StatusManager.UpdateStatus(f, "uploading", 60, fsha256)
				fcid, err := psfs.AddFileIPFS(ctx, folderPath+"/"+f)
				if err != nil {
					// Check if error is due to cancellation
					if ctx.Err() != nil {
						fmt.Println("[INFO] IPFS upload cancelled for file: " + f)
						store.StatusManager.FailUpload(f, "Upload cancelled during IPFS upload")
						return
					}
					fmt.Println("[ERROR] IPFS upload failed for file: "+f, err)
					store.StatusManager.FailUpload(f, "IPFS upload failed: "+err.Error())
					return
				}
				if fcid != "" {
					fmt.Println("[INFO] File: " + f + " ++added to IPFS with CID: " + fcid)
					store.StatusManager.UpdateStatus(f, "storing", 80, fsha256)

					fileExtension, err := psfs.GetExtension(f)
					if err != nil {
						store.StatusManager.FailUpload(f, "Failed to get file extension: "+err.Error())
						return
					}

					metadata := store.BaseMetadata{
						FileSHA256: strings.ToLower(fsha256),
						IPFSCID:    strings.ToLower(fcid),
						FileName:   f,
						FileType:   strings.ToLower(fileExtension),
					}

					errgs := store.GlobalStore.AddFile(metadata)
					if errgs != nil {
						fmt.Printf("[ERROR] failed to add file to GlobalStore: %w \n", errgs)
						store.StatusManager.FailUpload(f, "Failed to add to GlobalStore: "+errgs.Error())
						return
					}
					fmt.Println("[INFO] File: " + f + " ++added to GlobalStore with CID: " + fcid)

					// Save metadata immediately after successful upload
					store.GlobalStore.Save(appconfInstance.MetaDataFile)

					// Immediately publish metadata to P2P network for fast propagation
					go func(meta store.BaseMetadata) {
						if err := PublishMetadataUpdate(meta); err != nil {
							fmt.Printf("[ERROR] Failed to immediately publish metadata for %s: %v\n", meta.FileName, err)
						} else {
							fmt.Printf("[INFO] Successfully immediately published metadata for %s\n", meta.FileName)
						}
					}(metadata)

					store.StatusManager.CompleteUpload(f)
					if appconfInstance.FFMoveUpload {
						err := psfs.MoveFile(folderPath+"/"+f, appconfInstance.CacheFolder+"/"+f)
						if err != nil {
							fmt.Println("[ERROR] Error moving file: ", err)
						}
					}
				}
			} else {
				if appconfInstance.FFSendFileVT {
					// This was really to catch unknow files on VT

					// TODO: 				if appconfInstance.SecurityCapability [1 2 3 4]

					if appconfInstance.SecurityCapability == 4 {
						fmt.Println("[INFO] Submitting File to 3rd Party for Security check for file: " + f + " with SHA256: " + fsha256)
						submitresult, err := psfs.SendFileToVirusTotalWS(folderPath + "/" + f)
						if err != nil {
							fmt.Println("[ERROR] Error submitting file for security check: ", err)
							store.StatusManager.FailUpload(f, "Failed to submit to VirusTotal: "+err.Error())
						}
						if submitresult {
							fmt.Println("[INFO] Submission Passed Security check for file: " + f + " with SHA256: " + fsha256)
						} else {
							fmt.Println("[ERROR] File Security check failed for file: " + f + " with SHA256: " + fsha256)
							store.StatusManager.FailUpload(f, "Security check failed - potential malware detected")
							if appconfInstance.FFMoveUpload {
								err := psfs.MoveFile(folderPath+"/"+f, appconfInstance.RejectFolder+"/"+f)
								if err != nil {
									fmt.Println("[ERROR] Error moving file: ", err)
								}
							}
						}
					}
				} else {
					fmt.Println("[ERROR] File Security check failed for file: " + f + " with SHA256: " + fsha256)
					store.StatusManager.FailUpload(f, "Security check failed")
				}
			}
		} else {
			fmt.Println("[ERROR] File type invalid for file: " + f)
			store.StatusManager.FailUpload(f, "Invalid file type")
			if appconfInstance.FFMoveUpload {
				err := psfs.MoveFile(folderPath+"/"+f, appconfInstance.RejectFolder+"/"+f)
				if err != nil {
					fmt.Println("[ERROR] Error moving file: ", err)
				}
			}
			// move to rejected folder
			// log reason in rejected folder logfile
		}
	}
