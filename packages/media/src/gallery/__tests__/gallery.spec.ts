import { exampleUUID, createTestClient } from '@nice-cms/testing'
import { createGallery, Gallery } from '../gallery'

test('createGallery', async () => {
  const { client, mock } = createTestClient()

  const name = 'foo'
  mock.onPost('/galleries', { name }).reply(201, {
    id: exampleUUID,
    name: name,
    stacks: [],
  })

  let gallery: Gallery
  await expect(
    Promise.resolve().then(
      async () => (gallery = await createGallery(client, name))
    )
  ).resolves.not.toThrow()

  expect(gallery!.id).toBe(exampleUUID)
  expect(gallery!.name).toBe(name)
  expect(gallery!.stacks).toEqual([])
})
