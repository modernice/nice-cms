import { Image } from '../media'

/**
 * A gallery of images.
 */
export interface Gallery {
  id: string

  /**
   * Unique name of the gallery. No two galleries can have the same name.
   */
  name: string

  /**
   * Images of the gallery as stacks.
   */
  stacks: Stack[]
}

/**
 * A stack represents an image in one or many variants (e.g. different sizes).
 */
export interface Stack {
  id: string

  /**
   * Variants of the image.
   */
  images: StackImage[]
}

/**
 * An actual image of a stack.
 */
export interface StackImage extends Image {
  /**
   * Indicates whether the image is the original image of its stack.
   */
  original: boolean

  /**
   * Size of the image as a user-defined size key (e.g. "xs", "sm", "medium",
   * "highres" etc.). The original image has no size key (empty string).
   */
  size: string
}

/**
 * Hydrates an API response into a Gallery.
 */
export function hydrateGallery(data: any): Gallery {
  return {
    ...data,
    stacks: (data.stacks as any[]).map(hydrateStack),
  }
}

/**
 * Hydrates an API response into a Stack.
 */
export function hydrateStack(data: any): Stack {
  return {
    ...data,
    images: (data.images as any[]).map(hydrateStackImage),
  }
}

/**
 * Hydrates an API response into a StackImage.
 */
export function hydrateStackImage(data: any): StackImage {
  return {
    ...data,
  }
}
