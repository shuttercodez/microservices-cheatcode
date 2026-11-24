// https://nuxt.com/docs/api/configuration/nuxt-config
export default defineNuxtConfig({
  compatibilityDate: "2024-11-01",
  devtools: { enabled: false },
  modules: ["@nuxtjs/tailwindcss"],
  runtimeConfig: {
    public: {
      apiBaseUrl: process.env.NUXT_PUBLIC_API_BASE_URL || "http://localhost:8080",
    },
  },
  app: {
    head: {
      title: "Inventory & Order Microservice Dashboard (Vue 3 / Nuxt 3)",
      meta: [
        {
          name: "description",
          content: "Vue 3 + Nuxt 3 Frontend Dashboard for Go, Rust, and Python Microservices",
        },
      ],
    },
  },
});
