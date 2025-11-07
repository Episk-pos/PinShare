const Badge = ({ variant = 'default', children }) => {
  const base = 'inline-flex px-2 py-1 rounded-full text-xs font-medium'
  const variants = {
    green: 'bg-green-100 text-green-800',
    blue: 'bg-blue-100 text-blue-800',
    default: 'bg-gray-100 text-gray-800',
  }

  return (
    <span className={`${base} ${variants[variant] || variants.default}`}>
      {children}
    </span>
  )
}

export { Badge }