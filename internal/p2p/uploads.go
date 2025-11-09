package p2p

import (
	"fmt"
	"pinshare/internal/psfs"
	"pinshare/internal/store"
	"strings"
)

func ProcessUploads(folderPath string) {
	file, err := psfs.ListFiles(folderPath)
	var count int = 0
	if err != nil {
		return
	}
	for _, f := range file {
		// Start tracking this upload
		store.StatusManager.StartUpload(f)

		ftype, err := psfs.ValidateFileType(folderPath + "/" + f)
		if err != nil {
			fmt.Println("[ERROR] func ValidateFileType() error " + string(err.Error()))
			store.StatusManager.FailUpload(f, "File type validation failed: "+err.Error())
			continue
		}
		if ftype {
			fmt.Println("[INFO] File type valid for file: " + f)
			store.StatusManager.UpdateStatus(f, "hashing", 10, "")

			fsha256, err := psfs.GetSHA256(folderPath + "/" + f)
			if err != nil {
				fmt.Println("[ERROR] func GetSha256() error " + string(err.Error()))
				store.StatusManager.FailUpload(f, "Failed to calculate SHA256: "+err.Error())
				continue
			}

			var fresult bool
			if appconfInstance.FFIgnoreUploadsInMetadata {

				_, exists := store.GlobalStore.GetFile(fsha256)
				if exists {
					fmt.Printf("[WARNING] File already exists in GlobalStore with SHA256: %s \n", fsha256)
					store.StatusManager.FailUpload(f, "File already exists in GlobalStore")
					continue
				} else {

					if appconfInstance.SecurityCapability > 0 {
						fmt.Println("[INFO] File Security checking file: " + f + " with SHA256: " + fsha256)
						store.StatusManager.UpdateStatus(f, "scanning", 30, fsha256)

						var result bool
						var err error
						// TODO: 				if appconfInstance.SecurityCapability [1 2 3 4]
						if appconfInstance.SecurityCapability <= 3 {
							result, err = psfs.ClamScanFileClean(folderPath + "/" + f)
							if err != nil {
								fmt.Println("[ERROR] (ClamScanFileClean) " + string(err.Error()))
								store.StatusManager.FailUpload(f, "Security scan failed: "+err.Error())
								continue
							}
						}

						if appconfInstance.SecurityCapability == 4 {
							if appconfInstance.FFSkipVT {
								result = true
							} else {
								result, err = psfs.GetVirusTotalWSVerdictByHash(fsha256) // true == safe
								if err != nil {
									fmt.Println("[ERROR] (GetVirusTotalVerdictByHash) " + string(err.Error()))
									store.StatusManager.FailUpload(f, "VirusTotal scan failed: "+err.Error())
									continue
								}
							}
						}

						// fmt.Println("[INFOSEC] File Security check passed for file: " + f + " with SHA256: " + fsha256)
						fresult = result
					}

				}
			}

			if fresult {
				store.StatusManager.UpdateStatus(f, "uploading", 60, fsha256)
				fcid := psfs.AddFileIPFS(folderPath + "/" + f)
				if fcid != "" {
					fmt.Println("[INFO] File: " + f + " ++added to IPFS with CID: " + fcid)
					store.StatusManager.UpdateStatus(f, "storing", 80, fsha256)

					fileExtension, err := psfs.GetExtension(f)
					if err != nil {
						store.StatusManager.FailUpload(f, "Failed to get file extension: "+err.Error())
						continue
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
						continue
					}
					fmt.Println("[INFO] File: " + f + " ++added to GlobalStore with CID: " + fcid)
					store.StatusManager.CompleteUpload(f)
					count = count + 1
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
	if count >= 1 {
		store.GlobalStore.Save(appconfInstance.MetaDataFile)
	}
}
