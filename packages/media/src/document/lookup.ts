import type { AxiosInstance } from 'axios'

/**
 * Lookup the UUID of the shelf with the given name.
 *
 * @returns UUID of the shelf with that name
 */
export async function lookupShelfByName(client: AxiosInstance, name: string) {
  const { data } = await client.get(`/lookup/name/${name}`)
  return data.shelfId as string
}
