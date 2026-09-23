import { deliveryOptions, pickDelivery } from "../domain/deliveryModes.js";
import { api } from "./api.js";
import { subscribeFeed } from "./feed.js";

// The attach composer's delivery modes (ADR-0206): the CLI's modes from
// GET <base>/prompt once, and its state kept live from terminal.state.
// `base` is the door the composer sends to (/api/terminals/{id} or
// /api/agents/{id}). Each shell owns its presentation.
export function createUseDeliveryModes({ useEffect, useState }) {
  return function useDeliveryModes(base) {
    const [info, setInfo] = useState({ modes: [], state: "", termId: "" });
    const [pick, setPick] = useState("");

    useEffect(() => {
      let live = true;
      setInfo({ modes: [], state: "", termId: "" });
      if (!base) return undefined;
      // A failed read leaves the plain composer: Send stays a prompt and the
      // server's refusal still names what went wrong.
      api(base + "/prompt").then((d) => {
        if (live && d) setInfo({ modes: d.modes || [], state: d.state || "", termId: d.termId || "" });
      }).catch(() => {});
      return () => { live = false; };
    }, [base]);

    useEffect(() => {
      if (!info.termId) return undefined;
      return subscribeFeed((ev) => {
        if (!ev || ev.type !== "terminal.state" || !ev.data || ev.data.termId !== info.termId) return;
        const state = ev.data.state || "";
        setInfo((cur) => (cur.state === state ? cur : { ...cur, state }));
      });
    }, [info.termId]);

    const options = deliveryOptions(info.modes, info.state);
    return { options, delivery: pickDelivery(options, pick), setDelivery: setPick };
  };
}
