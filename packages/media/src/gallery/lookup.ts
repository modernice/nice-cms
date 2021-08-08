import { AxiosInstance } from 'axios'

/**
 * Lookup the UUID of the gallery with the given name.
 *
 * @returns UUID of the gallery with that name
 */
export async function lookupGalleryByName(client: AxiosInstance, name: string) {
  const { data } = await client.get(`/lookup/name/${name}`)
  return data.galleryId as string
}
