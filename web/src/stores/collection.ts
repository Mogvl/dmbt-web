// Collections store: IndexedDB-backed favorites mirroring
// apps/web/src/stores/collection.ts.

import { defineStore } from 'pinia';
import { openDB, type DBSchema, type IDBPDatabase } from 'idb';
import { generateCollection } from '../api/client';

export interface CollectionItem {
  name: string;
  searchParams: string;
  resources?: any[];
  complete?: boolean;
  [key: string]: any;
}

export interface Collection {
  hash?: string;
  name: string;
  authorization: string;
  filters: CollectionItem[];
}

const defaultCollection: Collection = {
  hash: undefined,
  name: '收藏夹',
  authorization: '',
  filters: []
};

const collectionsKey = 'animegarden:collections';
const collectionsDbVersion = 1;
const collectionsStoreName = 'key-value';
const currentCollectionNameKey = 'animegarden:cur_collection_name';

interface CollectionsDb extends DBSchema {
  [collectionsStoreName]: { key: string; value: Collection };
}

function readCurrentCollectionName(): string {
  try {
    return (JSON.parse(localStorage.getItem(currentCollectionNameKey) ?? '"收藏夹"') ??
      '收藏夹') as string;
  } catch {
    return '收藏夹';
  }
}

let collectionsDbPromise: Promise<IDBPDatabase<CollectionsDb>> | null = null;

function getDb() {
  if (!collectionsDbPromise) {
    collectionsDbPromise = openDB<CollectionsDb>(collectionsKey, collectionsDbVersion, {
      upgrade(database) {
        if (!database.objectStoreNames.contains(collectionsStoreName)) {
          database.createObjectStore(collectionsStoreName);
        }
      }
    });
  }
  return collectionsDbPromise;
}

export const useCollectionsStore = defineStore('collections', {
  state: () => ({
    collections: { 收藏夹: JSON.parse(JSON.stringify(defaultCollection)) } as Record<
      string,
      Collection
    >,
    currentCollectionName: readCurrentCollectionName() as string
  }),
  getters: {
    currentCollection(state): Collection | undefined {
      return state.collections[state.currentCollectionName];
    }
  },
  actions: {
    async load() {
      try {
        const db = await getDb();
        const [keys, values] = await Promise.all([
          db.getAllKeys(collectionsStoreName),
          db.getAll(collectionsStoreName)
        ]);
        const collections: Record<string, Collection> = {};
        keys.forEach((key, index) => {
          const value: unknown = values[index];
          if (value && typeof key === 'string' && Array.isArray((value as Collection).filters)) {
            collections[key] = value as Collection;
          }
        });
        if (Object.keys(collections).length > 0) {
          this.collections = collections;
        }
      } catch {
        // ignore
      }
    },
    async persist() {
      try {
        const db = await getDb();
        const tx = db.transaction(collectionsStoreName, 'readwrite');
        await Promise.all([
          ...Object.entries(this.collections).map(([key, value]: [string, Collection]) =>
            tx.store.put(value, key)
          )
        ]);
        await tx.done;
      } catch {
        // ignore
      }
    },
    setCurrentCollectionName(name: string) {
      this.currentCollectionName = name;
      localStorage.setItem(currentCollectionNameKey, JSON.stringify(name));
    },
    async addCollectionItem(item: CollectionItem, collectionName?: string) {
      const name = collectionName ?? this.currentCollectionName;
      const collection = this.collections[name] ?? {
        ...JSON.parse(JSON.stringify(defaultCollection)),
        name
      };
      collection.filters = [item, ...collection.filters];
      this.collections[name] = collection;
      await this.persist();
    },
    // Sync a filter item to the server collection (hash-based).
    async syncToServer() {
      const collection = this.currentCollection;
      if (!collection) return;
      const resp = await generateCollection({
        name: collection.name,
        authorization: collection.authorization,
        filters: collection.filters
      });
      if (resp.ok && resp.hash) {
        collection.hash = resp.hash;
        await this.persist();
      }
      return resp;
    },
    async removeCollectionItem(searchParams: string) {
      const collection = this.currentCollection;
      if (!collection) return;
      collection.filters = collection.filters.filter((f) => f.searchParams !== searchParams);
      await this.persist();
    },
    async updateCollectionItem(item: CollectionItem) {
      const collection = this.currentCollection;
      if (!collection) return;
      const idx = collection.filters.findIndex((f) => f.searchParams === item.searchParams);
      if (idx !== -1) {
        collection.filters[idx] = item;
        await this.persist();
      }
    }
  }
});
