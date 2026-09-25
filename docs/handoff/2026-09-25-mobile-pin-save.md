# 2026-09-25 — feat/mobile-pin-save: phone pin/snippet screens save from the form only
Shipped: owner asked to drop the duplicate Save on the phone. PinEdit and
SnippetEdit (web/mobile/src/screens/) lose the header "Save" (ScreenHeader
right slot); the form's button ("Create pin"/"Create snippet", "Save" when
editing) is the only save action and now carries the header's `!loaded` guard,
so an existing item cannot be saved before its server copy loads. No test or
QA script referenced the header button.
Verified: `make ci-scoped` PASS; scratch at 390px: no header Save on either
screen, Create pin from the form created and opened the pin. Blind spot: edit
screens of an existing item not captured; on a long body the form button sits
below the textarea (the header Save was always visible). visual-review: PASS
Not done / debts: none. Merge: landed on main 2b7aac93c; deployed by the owner 2026-09-25.
