'use client'

import { useEffect, useState, type ImgHTMLAttributes, type ReactNode } from 'react'

type SafeImageProps = ImgHTMLAttributes<HTMLImageElement> & {
  fallback?: ReactNode
  fallbackSrc?: string
}

export function SafeImage({ src, fallback, fallbackSrc, onError, ...props }: SafeImageProps) {
  const [currentSrc, setCurrentSrc] = useState(src)
  const [failed, setFailed] = useState(false)

  useEffect(() => {
    setCurrentSrc(src)
    setFailed(false)
  }, [src])

  if (failed) return fallback ? <>{fallback}</> : null

  return (
    <img
      {...props}
      src={currentSrc}
      onError={(event) => {
        onError?.(event)
        if (fallbackSrc && currentSrc !== fallbackSrc) {
          setCurrentSrc(fallbackSrc)
          return
        }
        setFailed(true)
      }}
    />
  )
}
