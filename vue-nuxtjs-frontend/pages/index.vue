<script setup lang="ts">
import { onMounted } from "vue";

const router = useRouter();
const { isAuthenticated, syncTokenFromStorage } = useConsole();

onMounted(() => {
  syncTokenFromStorage();
  router.replace(isAuthenticated.value ? "/products" : "/login");
});
</script>

<template>
  <div />
</template>
