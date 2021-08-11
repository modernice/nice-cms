import { createTestClient, exampleUUID, nilUUID } from '@nice-cms/testing'
import { lookupGalleryByName, lookupGalleryStackByName } from '../lookup'

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

describe('lookupGalleryStackByName', () => {
  it('returns the id and found=true if the id can be found', async () => {
    const { client, mock } = createTestClient()

    const name = 'foo'
    mock
      .onGet(`/galleries/${exampleUUID}/lookup/stack-name/${name}`)
      .reply(200, { stackId: exampleUUID })

    let id: string
    let found: boolean
    await expect(
      Promise.resolve().then(async () => {
        const resp = await lookupGalleryStackByName(client, exampleUUID, name)
        id = resp.id
        found = resp.found
      })
    ).resolves.not.toThrow()

    expect(id!).toBe(exampleUUID)
    expect(found!).toBe(true)
  })

  it('returns found=false if the id cannot be found', async () => {
    const { client, mock } = createTestClient()

    const name = 'foo'
    mock.onGet(`/galleries/${exampleUUID}/lookup/name/${name}`).reply(404)

    let id: string
    let found: boolean
    await expect(
      Promise.resolve().then(async () => {
        const resp = await lookupGalleryStackByName(client, exampleUUID, name)
        id = resp.id
        found = resp.found
      })
    ).resolves.not.toThrow()

    expect(id!).toBe(nilUUID)
    expect(found!).toBe(false)
  })
})
