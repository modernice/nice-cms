import { exampleUUID } from '@nice-cms/testing'
import { createClient } from '@nice-cms/core'
import { lookupShelfByName } from '../lookup'
import AxiosMock from 'axios-mock-adapter'

test('lookupShelfByName', async () => {
  const client = createClient('http://nice.test')
  const mock = new AxiosMock(client)

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
