import { exampleUUID } from '@nice-cms/testing'
import { ApiResponse } from '@nice-cms/core'
import {
  Gallery,
  hydrateStack,
  hydrateStackImage,
  Stack,
  StackImage,
} from './gallery'

export function makeGalleryResponse(
  options?: Partial<ApiResponse<Gallery>>
): ApiResponse<Gallery> {
  return {
    id: options?.id || exampleUUID,
    name: options?.name || 'foo',
    stacks: options?.stacks || [],
  }
}

export function makeStackResponse(
  options?: Partial<ApiResponse<Stack>>
): ApiResponse<Stack> {
  return {
    id: options?.id || exampleUUID,
    images: options?.images || [],
  }
}

export function makeImageResponse(options?: Partial<ApiResponse<StackImage>>) {
  return {
    disk: options?.disk || 'foo-disk',
    path: options?.path || '/foo.png',
    name: options?.name || 'foo',
    width: options?.width || 640,
    height: options?.height || 480,
    filesize: options?.filesize || 1234,
    original: options?.original !== void 0 ? options.original : true,
    size: options?.size || '',
    tags: options?.tags || [],
  }
}
