const palette = [
  "#e7000b",
  "#f54a00",
  "#e17100",
  "#d08700",
  "#5ea500",
  "#00a63e",
  "#009966",
  "#009689",
  "#0092b8",
  "#0084d1",
  "#155dfc",
  "#4f39f6",
  "#7f22fe",
  "#9810fa",
  "#c800de",
  "#e60076",
];
const productionId = "aaaaaaaaaaaaaaaaaaaa";
const stagingId = "bbbbbbbbbbbbbbbbbbbb";
const documentId = "cccccccccccccccccccc";
const mainBranchId = "dddddddddddddddddddd";
const draftBranchId = "eeeeeeeeeeeeeeeeeeee";
const documents = [
  {
    id: documentId,
    documentName: "Runbook",
    icon: "lucide:file",
    protected: false,
    children: null,
  },
];
const assignments = {
  [mainBranchId]: [productionId, stagingId],
  [draftBranchId]: [productionId],
};
const initialTags = [
  {
    id: productionId,
    tagName: "Production",
    color: "#00a63e",
    hidden: false,
    documents,
  },
  {
    id: stagingId,
    tagName: "Staging",
    color: "#f54a00",
    hidden: false,
    documents,
  },
];
let tags = structuredClone(initialTags);
let mode = "normal";
let unexpectedWrites: { method: string; path: string }[] = [];
let requests: {
  id: string;
  body: { tagName?: string; color?: string };
  outcome: string;
}[] = [];

export default defineEventHandler(async (event) => {
  const path = getRequestURL(event).pathname.replace(/^\/__tag_fixture/, "");
  setHeader(event, "Cache-Control", "no-store");

  if (event.method === "POST" && path === "/reset") {
    tags = structuredClone(initialTags);
    requests = [];
    unexpectedWrites = [];
    mode = "normal";
    return { ok: true };
  }
  if (event.method === "POST" && path === "/control") {
    const body = await readBody<{ mode?: string }>(event);
    if (!["normal", "fail-once", "slow-once"].includes(body.mode ?? "")) {
      throw createError({
        statusCode: 400,
        statusMessage: "Unknown fixture mode",
      });
    }
    mode = body.mode!;
    return { mode };
  }
  if (event.method === "PUT" && /^\/api\/tags\/[a-z0-9]{20}$/.test(path)) {
    const id = path.split("/").at(-1)!;
    const body = await readBody<{ tagName?: string; color?: string }>(event);
    const request = { id, body, outcome: "pending" };
    requests.push(request);
    const next = mode;
    mode = "normal";
    if (next === "slow-once")
      await new Promise((resolve) => setTimeout(resolve, 2000));
    const tag = tags.find((item) => item.id === id);
    let status = 204;
    let code = "";
    if (next === "fail-once") {
      status = 500;
      code = "fixture.save_failed";
    } else if (!tag) {
      status = 404;
      code = "tag.not_found";
    } else if (
      (body.tagName !== undefined && !body.tagName.trim()) ||
      (body.color !== undefined && !palette.includes(body.color))
    ) {
      status = 400;
      code = "fixture.invalid_input";
    } else if (
      body.tagName &&
      tags.some(
        (item) => item.id !== id && item.tagName === body.tagName!.trim(),
      )
    ) {
      status = 409;
      code = "tag.duplicate_name";
    }
    if (code) {
      request.outcome = code;
      setResponseStatus(event, status);
      return { code };
    }
    if (body.tagName !== undefined) tag!.tagName = body.tagName.trim();
    if (body.color !== undefined) tag!.color = body.color;
    request.outcome = "saved";
    setResponseStatus(event, 204);
    return null;
  }
  if (event.method !== "GET") {
    unexpectedWrites.push({ method: event.method, path });
    throw createError({
      statusCode: 405,
      statusMessage: "Unrelated fixture writes are disabled",
    });
  }
  if (path === "/state")
    return { tags, assignments, requests, unexpectedWrites, nextSave: mode };
  if (path.endsWith("/get-session"))
    return send(event, "null", "application/json");
  if (path.endsWith("/get-full-organization"))
    return {
      id: "fixture-org",
      name: "Tag validation",
      slug: "fixture",
      members: [{ id: "one" }, { id: "two" }],
    };
  if (path === "/api/tags/tree") return tags;
  if (path === "/api/documents/tree") return documents;
  const branch = path.match(
    /^\/api\/documents\/[^/]+\/branches\/([^/]+)\/tags$/,
  )?.[1];
  if (branch) return assignments[branch as keyof typeof assignments] ?? [];
  if (path === "/api/notifications/count") return { count: 0 };
  if (path === "/api/capabilities")
    return {
      github: false,
      slack: false,
      changeDetection: false,
      mcp: false,
      aiAssistant: { status: "inactive" },
    };
  if (path === "/api/github" || path === "/api/slack")
    return { configured: false, connected: false };
  return [];
});
