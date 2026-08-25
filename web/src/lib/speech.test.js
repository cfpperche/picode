import assert from "node:assert/strict";
import { test } from "node:test";
import {
  speechSupported, mergeTranscript, collectResults, humanizeSpeechError, discloseSttOnce,
} from "./speech.js";

test("speechSupported is false without a constructor", () => {
  assert.equal(speechSupported({}), false);
  assert.equal(speechSupported({ webkitSpeechRecognition: function Rec() {} }), true);
});

test("mergeTranscript joins with a single space", () => {
  assert.equal(mergeTranscript("", "ola"), "ola");
  assert.equal(mergeTranscript("ola", ""), "ola");
  assert.equal(mergeTranscript("ola  ", "  mundo"), "ola mundo");
  assert.equal(mergeTranscript("path /tmp", "file"), "path /tmp file");
});

test("collectResults splits final vs interim", () => {
  const results = [
    { isFinal: true, 0: { transcript: "hello " } },
    { isFinal: false, 0: { transcript: "world" } },
  ];
  assert.deepEqual(collectResults(results, 0), { final: "hello ", interim: "world" });
  assert.deepEqual(collectResults(results, 1), { final: "", interim: "world" });
});

test("humanizeSpeechError stays quiet on no-speech/abort", () => {
  assert.equal(humanizeSpeechError("no-speech"), "");
  assert.equal(humanizeSpeechError("aborted"), "");
  assert.match(humanizeSpeechError("not-allowed"), /denied/i);
  assert.match(humanizeSpeechError("not-supported"), /Chrome/i);
});

test("discloseSttOnce fires once", () => {
  const mem = new Map();
  const store = { getItem: (k) => mem.get(k) || null, setItem: (k, v) => mem.set(k, v) };
  const notes = [];
  assert.equal(discloseSttOnce((m) => notes.push(m), store), true);
  assert.equal(discloseSttOnce((m) => notes.push(m), store), false);
  assert.equal(notes.length, 1);
});
