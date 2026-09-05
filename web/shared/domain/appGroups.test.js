import test from "node:test";
import assert from "node:assert/strict";
import { readGroupPreferences, writeGroupPreferences, groupIsOpen, resetGroupSearch, toggleGroup } from "./appGroups.js";

const initial = (saved = {}) => ({ saved, search: { query: "", values: {} } });

test("groups start closed; saved folds survive refreshes and unrelated groups", () => {
  let state = initial();
  assert.equal(groupIsOpen(state, "bidwar", ""), false);
  state = toggleGroup(state, "bidwar", "");
  assert.equal(groupIsOpen(state, "bidwar", ""), true);
  assert.equal(groupIsOpen(state, "cognixse", ""), false);
  let stored;
  const storage = { setItem: (_key, value) => { stored = value; }, getItem: () => stored };
  writeGroupPreferences(state.saved, storage);
  state = initial(readGroupPreferences(storage));
  assert.equal(groupIsOpen(state, "bidwar", ""), true);
  state = toggleGroup(state, "bidwar", "");
  assert.equal(groupIsOpen(state, "bidwar", ""), false);
});

test("search reveals closed matches and allows folding without changing saved preferences", () => {
  let state = initial({ bidwar: false, cognixse: true });
  state = resetGroupSearch(state, " Postgres ");
  assert.equal(groupIsOpen(state, "bidwar", "postgres"), true);
  state = toggleGroup(state, "bidwar", "postgres");
  assert.equal(groupIsOpen(state, "bidwar", "postgres"), false);
  assert.equal(groupIsOpen(state, "cognixse", "postgres"), true);
  assert.deepEqual(state.saved, { bidwar: false, cognixse: true });
  state = resetGroupSearch(state, "auth");
  assert.equal(groupIsOpen(state, "bidwar", "auth"), true);
  state = resetGroupSearch(state, "");
  assert.equal(groupIsOpen(state, "bidwar", ""), false);
  assert.equal(groupIsOpen(state, "cognixse", ""), true);
  state = resetGroupSearch(state, "postgres");
  assert.equal(groupIsOpen(state, "bidwar", "postgres"), true);
});

test("bad or blocked storage leaves native groups usable", () => {
  for (const value of ["{", "null", "[]", "42"]) {
    assert.deepEqual(readGroupPreferences({ getItem: () => value }), {});
  }
  assert.deepEqual(readGroupPreferences({ getItem: () => '{"a":true,"b":false,"c":"true"}' }), { a: true, b: false });
  const blocked = { getItem: () => { throw Error("blocked"); }, setItem: () => { throw Error("blocked"); } };
  assert.deepEqual(readGroupPreferences(blocked), {});
  assert.doesNotThrow(() => writeGroupPreferences({ a: true }, blocked));
});
