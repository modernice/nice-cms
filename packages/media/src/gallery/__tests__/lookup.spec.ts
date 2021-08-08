import { createTestClient, exampleUUID } from '@nice-cms/testing'
import { lookupGalleryByName } from '../lookup'

test('lookupGalleryByName', async () => {
  const { client, mock } = createTestClient()

  const name = 'foo'
  mock.onGet(`/galleries/lookup/name/${name}`).reply(200, {
    galleryId: exampleUUID,
  })

  let result = ''
  await expect(
    Promise.resolve().then(
      async () => (result = await lookupGalleryByName(client, name))
    )
  ).resolves.not.toThrow()

  expect(result).toBe(exampleUUID)

  await expect(lookupGalleryByName(client, 'bar')).rejects.toThrow()
})
