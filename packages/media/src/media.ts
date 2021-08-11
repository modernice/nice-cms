import { ApiResponse } from '@nice-cms/core'

/**
 * A storage file.
 */
export interface StorageFile {
  /**
   * Name of the file.
   */
  name: string

  /**
   * Storage disk the file is stored in.
   */
  disk: string

  /**
   * Storage path.
   */
  path: string

  /**
   * Filesize in bytes.
   */
  filesize: number

  /**
   * Tags.
   */
  tags: string[]
}

/**
 * A storage image.
 */
export interface Image extends StorageFile {
  /**
   * Width in pixels.
   */
  width: number

  /**
   * Height in pixels.
   */
  height: number
}

/**
 * A storage document.
 */
export interface StorageDocument extends StorageFile {}

/**
 * Hydrates an API response into a File.
 */
export function hydrateStorageFile(
  data: ApiResponse<StorageFile>
): StorageFile {
  return { ...data }
}

/**
 * Hydrates an API response into an Image.
 */
export function hydrateStorageImage(data: ApiResponse<Image>): Image {
  return {
    ...data,
    ...hydrateStorageFile(data),
  }
}

/**
 * Hydrates an API response into a Document.
 */
export function hydrateStorageDocument(
  data: ApiResponse<StorageDocument>
): StorageDocument {
  return {
    ...data,
    ...hydrateStorageFile(data),
  }
}
