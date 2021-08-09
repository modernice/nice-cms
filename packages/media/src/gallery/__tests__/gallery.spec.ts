import { exampleUUID, createTestClient, randomUUID } from '@nice-cms/testing'
import {
  createGallery,
  deleteGalleryStack,
  fetchGallery,
  Gallery,
  hydrateGallery,
  hydrateStack,
  hydrateStackImage,
  Stack,
  tagGalleryStack,
  untagGalleryStack,
  updateGalleryStack,
} from '../gallery'
import {
  makeGalleryResponse,
  makeImageResponse,
  makeStackResponse,
} from '../test-utils'

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

test('fetchGallery', async () => {
  const { client, mock } = createTestClient()

  const galleryData = makeGalleryResponse({
    stacks: [makeStackResponse({ images: [makeImageResponse()] })],
  })
  const want = hydrateGallery(galleryData)

  mock.onGet(`/galleries/${exampleUUID}`).reply(200, galleryData)

  let gallery: Gallery
  await expect(
    Promise.resolve().then(
      async () => (gallery = await fetchGallery(client, want.id))
    )
  ).resolves.not.toThrow()

  expect(gallery!).toEqual(want)
})

test('updateGalleryStack', async () => {
  const { client, mock } = createTestClient()

  const gallery = hydrateGallery(
    makeGalleryResponse({
      stacks: [
        makeStackResponse({
          images: [
            makeImageResponse({ name: 'bar', tags: ['foo', 'bar', 'baz'] }),
          ],
        }),
      ],
    })
  )

  const name = 'foo'
  const imageResponse = makeImageResponse({ name, tags: ['foo', 'bar', 'baz'] })
  const response = makeStackResponse({ images: [imageResponse] })
  mock
    .onPatch(`/galleries/${exampleUUID}/stacks/${exampleUUID}`)
    .reply(200, response)

  let updated: Stack
  await expect(
    Promise.resolve().then(
      async () =>
        (updated = await updateGalleryStack(client, gallery, exampleUUID, {
          name,
        }))
    )
  ).resolves.not.toThrow()

  expect(updated!.id).toBe(exampleUUID)
  expect(updated!.images).toEqual([hydrateStackImage(imageResponse)])
  expect(gallery.stacks).toContainEqual(updated!)
})

test('deleteGalleryStack', async () => {
  const { client, mock } = createTestClient()

  const stackId = randomUUID()
  const stackData = makeStackResponse({ id: stackId })
  const stack = hydrateStack(stackData)
  const gallery = hydrateGallery(
    makeGalleryResponse({ stacks: [makeStackResponse(), stackData] })
  )

  mock.onDelete(`/galleries/${gallery.id}/stacks/${stackId}`).reply(204)

  await expect(
    deleteGalleryStack(client, gallery, stackId)
  ).resolves.not.toThrow()

  expect(gallery.stacks).not.toContainEqual(stack)
})

test('tagGalleryStack', async () => {
  const { client, mock } = createTestClient()

  const stackData = makeStackResponse()
  const stack = hydrateStack(stackData)
  const gallery = hydrateGallery(makeGalleryResponse({ stacks: [stackData] }))

  const tags = ['foo', 'foo', 'bar', 'baz', 'baz']
  const wantTags = ['foo', 'bar', 'baz']

  const taggedStackData = makeStackResponse({
    images: [makeImageResponse({ tags: wantTags })],
  })
  const wantTaggedStack = hydrateStack(taggedStackData)

  mock
    .onPost(`/galleries/${gallery.id}/stacks/${stack.id}/tags`, { tags })
    .reply(201, taggedStackData)

  let taggedStack: Stack
  await expect(
    Promise.resolve().then(
      async () =>
        (taggedStack = await tagGalleryStack(client, gallery, stack.id, tags))
    )
  ).resolves.not.toThrow()

  expect(taggedStack!).toEqual(wantTaggedStack)
})

test('untagGalleryStack', async () => {
  const { client, mock } = createTestClient()

  const stackData = makeStackResponse({
    images: [
      makeImageResponse({
        tags: ['foo', 'bar', 'baz'],
      }),
    ],
  })
  const stack = hydrateStack(stackData)
  const gallery = hydrateGallery(makeGalleryResponse({ stacks: [stackData] }))

  const tagsToDelete = ['foo', 'baz']

  const stackResponse = makeStackResponse({
    images: [makeImageResponse({ tags: ['bar'] })],
  })

  mock
    .onDelete(
      `/galleries/${gallery.id}/stacks/${stack.id}/tags/${tagsToDelete.join(
        ','
      )}`
    )
    .reply(200, stackResponse)

  let untagged: Stack
  await expect(
    Promise.resolve().then(
      async () =>
        (untagged = await untagGalleryStack(client, gallery, stack.id, [
          'foo',
          'baz',
        ]))
    )
  ).resolves.not.toThrow()

  expect(untagged!.images[0].tags).toEqual(['bar'])
  expect(gallery.stacks).toContainEqual(untagged!)
  expect(gallery.stacks).not.toContainEqual(stack)
})
