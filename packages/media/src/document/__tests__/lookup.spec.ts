import { createTestClient, exampleUUID } from '@nice-cms/testing'
import { lookupShelfByName } from '../lookup'

test('lookupShelfByName', async () => {
  const { client, mock } = createTestClient()

  const name = 'foo'
  mock.onGet(`/lookup/name/${name}`).reply(200, {
    shelfId: exampleUUID,
  })

  let id = ''
  await expect(
    Promise.resolve().then(
      async () => (id = await lookupShelfByName(client, 'foo'))
    )
  ).resolves.not.toThrow()

  expect(id).toBe(exampleUUID)

  await expect(lookupShelfByName(client, 'bar')).rejects.toThrow()
})
