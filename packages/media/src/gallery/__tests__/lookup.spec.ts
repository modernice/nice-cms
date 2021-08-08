import { exampleUUID } from '@nice-cms/testing'
import { createClient } from '@nice-cms/core'
import { lookupGalleryByName } from '../lookup'
import AxiosMock from 'axios-mock-adapter'

test('lookupGalleryByName', async () => {
  const client = createClient('http://nice.test')
  const mock = new AxiosMock(client)

  const name = 'foo'
  mock.onGet(`/lookup/name/${name}`).reply(200, {
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
