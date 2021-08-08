import { createClient } from '../client'

test('createClient', () => {
  let client = createClient('http://nice.test/api')
  expect(client.defaults.baseURL).toBe('http://nice.test/api')
  expect(client.defaults.headers['Content-Type']).toBe('application/json')

  client = createClient('http://nice.test', {
    axios: { headers: { 'x-foo': 'bar' } },
  })

  expect(client.defaults.headers['x-foo']).toBe('bar')

  client = createClient('http://nice.test', {
    axios: {
      headers: { 'Content-Type': 'text/html' },
    },
  })

  expect(client.defaults.headers['Content-Type']).toBe('application/json')
})
