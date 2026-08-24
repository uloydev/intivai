import { beforeEach, describe, expect, it } from "vitest"
import { getStoredInvitationToken, storeInvitationToken } from "./interview-ticket"

describe("invitation ticket store", () => {
  beforeEach(() => sessionStorage.clear())

  it("stores and returns the invitation token keyed by interview id", () => {
    storeInvitationToken("iv-1", "inv-tok-1")
    expect(getStoredInvitationToken("iv-1")).toBe("inv-tok-1")
    expect(getStoredInvitationToken("iv-2")).toBeNull()
  })

  it("overwrites a previous token for the same interview", () => {
    storeInvitationToken("iv-1", "old")
    storeInvitationToken("iv-1", "new")
    expect(getStoredInvitationToken("iv-1")).toBe("new")
  })

  it("returns null when nothing was stored or the value is corrupt", () => {
    expect(getStoredInvitationToken("missing")).toBeNull()
    sessionStorage.setItem("intivai_invitation_iv-x", "not-json")
    expect(getStoredInvitationToken("iv-x")).toBeNull()
  })
})
