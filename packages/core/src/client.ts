import Axios, { AxiosRequestConfig } from 'axios'

/**
 * Creates the nice-cms client.
 *
 * @param baseUrl Base URL of the nice-cms API
 * @param options Client options
 */
export function createClient(
  baseUrl: string,
  options?: {
    /**
     * Custom axios configuration.
     */
    axios?: AxiosRequestConfig
  }
) {
  options = options || {}
  options.axios = options.axios || {}
  options.axios.headers = options.axios.headers || {}
  options.axios.headers['Content-Type'] = 'application/json'

  return Axios.create({
    ...options.axios,
    baseURL: baseUrl,
  })
}
