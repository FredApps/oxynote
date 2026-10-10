<script setup lang="ts">
import AppSidebar from "~/components/AppSidebar.vue";
import TagEditModal from "~/components/TagEditModal.vue";
import DocumentTagList from "~/components/editor/DocumentTagList.vue";
import {
  SidebarProvider,
  SidebarTrigger,
} from "~/components/shadcn/ui/sidebar";

const editor = useEditorStore();
const { setEditable } = useEditorMeta();
const { fetchTagTree } = useTagAPI();
const pending = ref<Pick<TagTreeElement, "id" | "tagName" | "color"> | null>(
  null,
);
const snapshot = ref<unknown>();
const message = ref("");
const nextSave = ref("normal");
const branch = ref("main");

editor.updateActiveDocumentId("cccccccccccccccccccc");
editor.updateActiveBranchId("dddddddddddddddddddd");
editor.setActiveBranchProtected(false);
editor.setReviewableDiffActive(false);
editor.updateTargetBranchId(null);
setEditable(true);

onMounted(refreshSnapshot);
watch(() => fetchTagTree.data.value, refreshSnapshot);
watch(pending, (value) => {
  if (!value) void refreshSnapshot();
});
watch(branch, (value) => {
  editor.updateActiveBranchId(
    value === "main" ? "dddddddddddddddddddd" : "eeeeeeeeeeeeeeeeeeee",
  );
});

async function refreshSnapshot() {
  snapshot.value = await $fetch("/__tag_fixture/state");
}

async function armSave() {
  await $fetch("/__tag_fixture/control", {
    method: "POST",
    body: { mode: nextSave.value },
  });
  message.value = `Next save: ${nextSave.value}`;
}

async function reset() {
  await $fetch("/__tag_fixture/reset", { method: "POST" });
  window.location.reload();
}
</script>

<template>
  <SidebarProvider :default-open="true">
    <AppSidebar
      :all-initial-sections-loaded="true"
      :notification-sidebar-open="false"
      @edit-tag="pending = $event"
    />
    <main class="min-w-0 flex-1 bg-background p-6 text-foreground">
      <header class="mb-6 flex items-center gap-3">
        <SidebarTrigger />
        <h1 class="text-xl font-semibold">
          Oxynote #94: tag editing validation
        </h1>
      </header>
      <p class="mb-4 text-sm text-muted-foreground">
        Real sidebar, edit modal, API composable and document pills. Synthetic
        in-memory API only; no account, database or websocket persistence.
      </p>
      <section class="mb-6 rounded border p-4">
        <h2 class="mb-3 font-semibold">Runbook document</h2>
        <label class="mb-4 flex items-center gap-2 text-sm">
          Active branch
          <select v-model="branch" class="rounded border bg-background p-2">
            <option value="main">Main (two assigned tags)</option>
            <option value="draft">Draft (first tag only)</option>
          </select>
        </label>
        <DocumentTagList />
      </section>
      <section class="mb-6 flex flex-wrap items-end gap-3 rounded border p-4">
        <label class="flex flex-col gap-2 text-sm">
          Next save behavior
          <select v-model="nextSave" class="rounded border bg-background p-2">
            <option value="normal">Normal</option>
            <option value="fail-once">Fail once (retry succeeds)</option>
            <option value="slow-once">Slow once (2 seconds)</option>
          </select>
        </label>
        <button class="rounded border px-3 py-2" @click="armSave">
          Arm next save
        </button>
        <button class="rounded border px-3 py-2" @click="refreshSnapshot">
          Refresh audit
        </button>
        <button class="rounded border px-3 py-2" @click="reset">
          Reset fixture
        </button>
        <p class="w-full text-sm" role="status">{{ message }}</p>
      </section>
      <p class="mb-3 text-sm">
        Use the tag row's menu → Edit Tag. Renaming Production to Staging
        reproduces a duplicate-name response. Check both branches after saving.
      </p>
      <pre
        class="overflow-auto rounded border p-4 text-xs"
        data-testid="fixture-audit"
        >{{ JSON.stringify(snapshot, null, 2) }}</pre>
    </main>
    <TagEditModal v-model="pending" />
  </SidebarProvider>
</template>
