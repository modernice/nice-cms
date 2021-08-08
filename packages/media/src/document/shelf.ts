/**
 * A shelf of documents.
 */
export interface Shelf {
  id: string

  /**
   * Name of the shelf.
   */
  name: string

  /**
   * Documents of this shelf.
   */
  documents: ShelfDocument[]
}

/**
 * A document within a shelf.
 */
export interface ShelfDocument extends Document {
  id: string

  /**
   * Unique name of the document. This name is unique within the shelf of the
   * document. Uniqueness across multiple shelfs is not guaranteed.
   */
  uniqueName: string
}
