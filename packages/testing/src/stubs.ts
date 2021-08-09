export const exampleUUID = 'b53abd66-e8a5-4962-9a94-586af698d358'

export function randomUUID() {
  const crypto = globalThis.crypto || require('crypto').webcrypto

  // @ts-ignore
  return ([1e7] + -1e3 + -4e3 + -8e3 + -1e11).replace(/[018]/g, (c) =>
    (
      c ^
      (crypto.getRandomValues(new Uint8Array(1))[0] & (15 >> (c / 4)))
    ).toString(16)
  )
}
