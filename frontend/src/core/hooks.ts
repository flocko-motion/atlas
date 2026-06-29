/**
 * @layer core
 * @description React bindings for the vanilla Zustand store.
 * @depends core/store
 * @must-not Contain business logic. Only re-exports store as React hook.
 */

import { useStore } from 'zustand';
import { store, type AppState } from './store';

export function useAppStore<T>(selector: (state: AppState) => T): T {
  return useStore(store, selector);
}
