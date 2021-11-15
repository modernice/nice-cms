import { AxiosInstance } from 'axios'
import { ApiResponse } from '@nice-cms/core'
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
export function hydrateGallery(data: ApiResponse<Gallery>): Gallery {
  return {
    ...data,
    stacks: (data.stacks as any[]).map(hydrateStack),
  }
}

/**
 * Hydrates an API response into a Stack.
 */
export function hydrateStack(data: ApiResponse<Stack>): Stack {
  return {
    ...data,
    images: data.images.map(hydrateStackImage),
  }
}

/**
 * Hydrates an API response into a StackImage.
 */
export function hydrateStackImage(data: ApiResponse<StackImage>): StackImage {
  return {
    ...data,
  }
}

/**
 * Create a new gallery with the given name.
 */
export async function createGallery(client: AxiosInstance, name: string) {
  const { data } = await client.post('/galleries', { name })
  return hydrateGallery(data)
}

/**
 * Fetch the gallery with the given UUID.
 */
export async function fetchGallery(client: AxiosInstance, id: string) {
  const { data } = await client.get(`/galleries/${id}`)
  return hydrateGallery(data)
}

/**
 * Uploads an image into a gallery and returns the created stack. The stack is
 * pushed into the provided gallery's stacks.
 */
export async function uploadToGallery(
  client: AxiosInstance,
  gallery: Gallery,
  image: File,
  name: string,
  disk: string,
  path: string
) {
  const formData = new FormData()
  formData.append('image', image)
  formData.append('name', name)
  formData.append('disk', disk)
  formData.append('path', path)

  const { data } = await client.post(
    `/galleries/${gallery.id}/stacks`,
    formData
  )
  const stack = hydrateStack(data)

  gallery.stacks.push(stack)

  return stack
}

/**
 * Replaces the image of the given stack and returns the updated stack. The
 * stack in the gallery is replaced with that stack.
 */
export async function replaceGalleryImage(
  client: AxiosInstance,
  gallery: Gallery,
  stackId: string,
  image: File
) {
  const formData = new FormData()
  formData.append('image', image)

  const { data } = await client.put(
    `/galleries/${gallery.id}/stacks/${stackId}`,
    formData
  )
  const stack = hydrateStack(data)

  gallery.stacks.splice(
    gallery.stacks.findIndex((s) => s.id === stackId),
    1,
    stack
  )

  return stack
}

/**
 * Updates the stack with the given stackId and returns the updated stack.
 */
export async function updateGalleryStack(
  client: AxiosInstance,
  gallery: Gallery,
  stackId: string,
  options: {
    name?: string
  }
) {
  const { data } = await client.patch(
    `/galleries/${gallery.id}/stacks/${stackId}`,
    {
      name: options.name,
    }
  )

  const stack = hydrateStack(data)
  gallery.stacks.splice(
    gallery.stacks.findIndex((s) => s.id === stackId),
    1,
    stack
  )

  return stack
}

/**
 * Deletes the stack with the given stackId from the gallery and returns the
 * deleted stack.
 */
export async function deleteGalleryStack(
  client: AxiosInstance,
  gallery: Gallery,
  stackId: string
) {
  await client.delete(`/galleries/${gallery.id}/stacks/${stackId}`)
  const deleted = gallery.stacks.splice(
    gallery.stacks.findIndex((s) => s.id === stackId),
    1
  )
  if (!deleted.length) {
    throw new Error(`Stack "${stackId}" not removed from stacks array.`)
  }
  return deleted[0]
}

/**
 * Adds tags to the images of a gallery's stack.
 */
export async function tagGalleryStack(
  client: AxiosInstance,
  gallery: Gallery,
  stackId: string,
  tags: string[]
) {
  const { data } = await client.post(
    `/galleries/${gallery.id}/stacks/${stackId}/tags`,
    { tags }
  )
  return hydrateStack(data)
}

/**
 * Removes tags from the images of a gallery's stack.
 */
export async function untagGalleryStack(
  client: AxiosInstance,
  gallery: Gallery,
  stackId: string,
  tags: string[]
) {
  const { data } = await client.delete(
    `/galleries/${gallery.id}/stacks/${stackId}/tags/${tags.join(',')}`
  )
  const stack = hydrateStack(data)
  gallery.stacks.splice(
    gallery.stacks.findIndex((s) => s.id === stackId),
    1,
    stack
  )
  return stack
}

/**
 * Sorts the given {@link Gallery} by the provided `sorting`.
 */
export async function sortGallery(
  client: AxiosInstance,
  gallery: Gallery,
  sorting: string[]
) {
  await client.patch(`/galleries/${gallery.id}/sorting`, { sorting })
  const update = await fetchGallery(client, gallery.id)
  gallery.stacks = update.stacks
}

/**
 * Returns the stack with the given stackId from the stacks of the gallery.
 */
export function findGalleryStack(gallery: Gallery, stackId: string) {
  return gallery.stacks.find((stack) => stack.id === stackId)
}

/**
 * Returns the first image of the stack with the given size name.
 */
export function findStackImageBySizeName(
  stack: Stack,
  size: string,
  options?: {
    fallbackOriginal: boolean
  }
) {
  const img = stack.images.find((img) => img.size === size)
  if (!img && options?.fallbackOriginal) {
    return findOriginalStackImage(stack)
  }
  return img
}

/**
 * Returns the first image of the stack with the given width.
 */
export function findStackImageByWidth(
  stack: Stack,
  width: number,
  options?: {
    fallbackOriginal: boolean
  }
) {
  const img = stack.images.find((img) => img.width === width)
  if (!img && options?.fallbackOriginal) {
    return findOriginalStackImage(stack)
  }
  return img
}

/**
 * Returns the first image of the stack with the given height.
 */
export function findStackImageByHeight(
  stack: Stack,
  height: number,
  options?: {
    fallbackOriginal: boolean
  }
) {
  const img = stack.images.find((img) => img.height === height)
  if (!img && options?.fallbackOriginal) {
    return findOriginalStackImage(stack)
  }
  return img
}

/**
 * Returns the original image of the stack.
 */
export function findOriginalStackImage(stack: Stack) {
  for (const img of stack.images) {
    if (img.original) {
      return img
    }
  }
}
