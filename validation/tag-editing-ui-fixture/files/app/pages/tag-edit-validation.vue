<script setup lang="ts">
definePageMeta({ skipAuth: true });
</script>

<template>
  <ClientOnly>
    <TagEditValidation />
  </ClientOnly>
</template>
