import Axios, { AxiosRequestConfig } from 'axios'

export type ApiResponse<T = any> = T extends { [key: string]: any }
  ? { [K in keyof T]: ApiResponse<T[K]> }
  : T extends string
  ? T
  : T extends number
  ? T
  : T extends boolean
  ? T
  : any

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
