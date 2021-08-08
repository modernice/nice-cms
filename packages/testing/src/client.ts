import { createClient } from '@nice-cms/core'
import AxiosMock from 'axios-mock-adapter'

export function createTestClient() {
  const client = createClient('http://nice.test')
  const mock = new AxiosMock(client)

  return {
    client,
    mock,
  }
}
