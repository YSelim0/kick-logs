import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { RequestAdmin } from "@/features/requests/request-admin";
import type { UserRequest, UserRequestDetailResponse, UserRequestEvent } from "@/types/api";

const apiMocks = vi.hoisted(() => ({
  addUserRequestNote: vi.fn(),
  archiveUserRequest: vi.fn(),
  getUserRequest: vi.fn(),
  listUserRequests: vi.fn(),
  updateUserRequestStatus: vi.fn()
}));

vi.mock("@/features/requests/api", () => ({
  addUserRequestNote: apiMocks.addUserRequestNote,
  archiveUserRequest: apiMocks.archiveUserRequest,
  getUserRequest: apiMocks.getUserRequest,
  listUserRequests: apiMocks.listUserRequests,
  updateUserRequestStatus: apiMocks.updateUserRequestStatus
}));

describe("RequestAdmin", () => {
  beforeEach(() => {
    apiMocks.addUserRequestNote.mockReset();
    apiMocks.archiveUserRequest.mockReset();
    apiMocks.getUserRequest.mockReset();
    apiMocks.listUserRequests.mockReset();
    apiMocks.updateUserRequestStatus.mockReset();

    apiMocks.listUserRequests.mockResolvedValue({ items: [requestFixture()], count: 1 });
    apiMocks.getUserRequest.mockResolvedValue(detailFixture());
    apiMocks.updateUserRequestStatus.mockResolvedValue(
      detailFixture({
        request: { ...requestFixture(), current_status: "approved" },
        events: [eventFixture("evt_status", "status_changed", "approved")]
      })
    );
    apiMocks.addUserRequestNote.mockResolvedValue(
      detailFixture({
        events: [eventFixture("evt_note", "note_added", "", "Kontrol edildi.")]
      })
    );
    apiMocks.archiveUserRequest.mockResolvedValue(
      detailFixture({
        request: { ...requestFixture(), is_archived: true },
        events: [eventFixture("evt_archive", "archived")]
      })
    );
  });

  it("loads active requests by default", async () => {
    render(<RequestAdmin />);

    await waitFor(() => expect(apiMocks.listUserRequests).toHaveBeenCalledTimes(1));
    expect(apiMocks.listUserRequests).toHaveBeenCalledWith({
      archived: false,
      limit: 50,
      q: undefined,
      start: undefined,
      end: undefined,
      status: undefined,
      type: undefined
    });
    expect(await screen.findAllByText("Kanal takip edilsin")).not.toHaveLength(0);
    expect(screen.getByText("#samplechannel · mod@example.com")).toBeInTheDocument();
  });

  it("applies request filters", async () => {
    const user = userEvent.setup();
    render(<RequestAdmin />);

    await waitFor(() => expect(apiMocks.listUserRequests).toHaveBeenCalledTimes(1));

    await user.selectOptions(screen.getByLabelText("Tip"), "feedback");
    await user.selectOptions(screen.getByLabelText("Durum"), "reviewing");
    await user.selectOptions(screen.getByLabelText("Arşiv"), "all");
    await user.type(screen.getByLabelText("Arama"), "export");
    await user.click(screen.getByRole("button", { name: "Filtrele" }));

    await waitFor(() => expect(apiMocks.listUserRequests).toHaveBeenCalledTimes(2));
    expect(apiMocks.listUserRequests).toHaveBeenLastCalledWith({
      archived: undefined,
      limit: 50,
      q: "export",
      start: undefined,
      end: undefined,
      status: "reviewing",
      type: "feedback"
    });
  });

  it("loads request detail and updates status", async () => {
    const user = userEvent.setup();
    render(<RequestAdmin />);

    await clickRequestRow(user);

    await waitFor(() => expect(apiMocks.getUserRequest).toHaveBeenCalledWith("req_1"));
    expect(await screen.findByText("Bu kanal eklenebilir mi?")).toBeInTheDocument();

    await user.selectOptions(screen.getByDisplayValue("Yeni"), "approved");
    await user.click(screen.getByRole("button", { name: "Kaydet" }));

    await waitFor(() =>
      expect(apiMocks.updateUserRequestStatus).toHaveBeenCalledWith("req_1", {
        status: "approved"
      })
    );
    expect(await screen.findByText("Durum: Onaylandı")).toBeInTheDocument();
  });

  it("adds a note and archives the selected request", async () => {
    const user = userEvent.setup();
    render(<RequestAdmin />);

    await clickRequestRow(user);
    await screen.findByText("Bu kanal eklenebilir mi?");

    await user.type(screen.getByPlaceholderText("İnceleme notu ekle"), "Kontrol edildi.");
    await user.click(screen.getByRole("button", { name: "Not ekle" }));

    await waitFor(() =>
      expect(apiMocks.addUserRequestNote).toHaveBeenCalledWith("req_1", {
        note: "Kontrol edildi."
      })
    );
    expect(await screen.findByText("Not eklendi")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Arşivle" }));

    await waitFor(() => expect(apiMocks.archiveUserRequest).toHaveBeenCalledWith("req_1"));
    expect(await screen.findAllByText("Arşiv")).not.toHaveLength(0);
  });
});

function requestFixture(overrides: Partial<UserRequest> = {}): UserRequest {
  return {
    request_id: "req_1",
    type: "channel_request",
    title: "Kanal takip edilsin",
    message: "Bu kanal eklenebilir mi?",
    channel_slug: "samplechannel",
    channel_display_name: "samplechannel",
    contact: "mod@example.com",
    current_status: "new",
    is_archived: false,
    created_at: "2026-06-14T09:00:00Z",
    latest_event_at: "2026-06-14T09:00:00Z",
    ...overrides
  };
}

function eventFixture(
  eventID: string,
  eventType: UserRequestEvent["event_type"],
  status: UserRequestEvent["status"] = "",
  note = ""
): UserRequestEvent {
  return {
    event_id: eventID,
    request_id: "req_1",
    event_type: eventType,
    status,
    note,
    admin_id: 1,
    created_at: "2026-06-14T09:05:00Z"
  };
}

function detailFixture(
  overrides: Partial<UserRequestDetailResponse> = {}
): UserRequestDetailResponse {
  return {
    request: requestFixture(),
    events: [],
    ...overrides
  };
}

async function clickRequestRow(user: ReturnType<typeof userEvent.setup>) {
  const titles = await screen.findAllByText("Kanal takip edilsin");
  const rowButton = titles[0].closest("button");
  if (!rowButton) {
    throw new Error("request row button not found");
  }
  await user.click(rowButton);
}
