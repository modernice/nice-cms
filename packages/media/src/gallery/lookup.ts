import { nilUUID } from '@nice-cms/testing'
import { AxiosError, AxiosInstance } from 'axios'

/**
 * Lookup the UUID of the gallery with the given name.
 *
 * @returns UUID of the gallery with that name
 */
export async function lookupGalleryByName(client: AxiosInstance, name: string) {
  const { data } = await client.get(`/galleries/lookup/name/${name}`)
  return data.galleryId as string
}

export async function lookupGalleryStackByName(
  client: AxiosInstance,
  name: string
) {
  try {
    const { data } = await client.get(`/galleries/lookup/name/${name}`)
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
