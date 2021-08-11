import { nilUUID } from '@nice-cms/testing'
import { AxiosError, AxiosInstance } from 'axios'

/**
 * Looks up the UUID of the gallery with the given name.
 */
export async function lookupGalleryByName(client: AxiosInstance, name: string) {
  const { data } = await client.get(`/galleries/lookup/name/${name}`)
  return data.galleryId as string
}

/**
 * Looks up the UUID of the stack with the given name within the given gallery.
 */
export async function lookupGalleryStackByName(
  client: AxiosInstance,
  galleryId: string,
  name: string
) {
  try {
    const { data } = await client.get(
      `/galleries/${galleryId}/lookup/stack-name/${name}`
    )
    return {
      id: data.stackId as string,
      found: true,
    }
  } catch (e) {
    const err = e as AxiosError

    if (err.response?.status === 404) {
      return {
        id: nilUUID,
        found: false,
      }
    }

    throw e
  }
}
