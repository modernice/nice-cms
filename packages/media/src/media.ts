import { ApiResponse } from '@nice-cms/core'

/**
 * A storage file.
 */
export interface File {
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
export interface Image extends File {
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
export interface Document extends File {}

/**
 * Hydrates an API response into a File.
 */
export function hydrateFile(data: ApiResponse<File>): File {
  return { ...data }
}

/**
 * Hydrates an API response into an Image.
 */
export function hydrateImage(data: ApiResponse<Image>): Image {
  return {
    ...data,
    ...hydrateFile(data),
  }
}

/**
 * Hydrates an API response into a Document.
 */
export function hydrateDocument(data: ApiResponse<Document>): Document {
  return {
    ...data,
    ...hydrateFile(data),
  }
}
