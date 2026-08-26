/**
 * package: core / tests
 * type:    test
 * job:     pin the claim-bytes cache — what it remembers, and that a session reset empties it
 * limits:  headless; the module-scope cache alone, no fetch and no store
 */

import { test } from 'node:test';
import assert from 'node:assert/strict';

import { cachedClaimBytesCount, claimBytesOf, forgetClaimBytes, rememberClaimBytes } from './claimBytes.ts';

test('a remembered claim is held until a session reset forgets it', () => {
  forgetClaimBytes();
  assert.equal(cachedClaimBytesCount(), 0);
  assert.equal(claimBytesOf('claim-a'), null);

  rememberClaimBytes('claim-a', new Uint8Array([1, 2, 3]));
  assert.equal(cachedClaimBytesCount(), 1);
  assert.deepEqual(claimBytesOf('claim-a'), new Uint8Array([1, 2, 3]));

  forgetClaimBytes();
  assert.equal(cachedClaimBytesCount(), 0, 'a session reset left claims cached');
  assert.equal(claimBytesOf('claim-a'), null, 'a session reset left a claim readable');
});
