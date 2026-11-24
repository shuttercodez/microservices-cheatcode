<script setup lang="ts">
import { ref } from "vue";
import type { CreateProductRequest } from "~/types/product";

const emit = defineEmits<{
  (e: "create", req: CreateProductRequest): void;
}>();

const sku = ref("");
const name = ref("");
const price = ref<number | string>("");
const stock = ref<number | string>("");

function handleSubmit() {
  emit("create", {
    sku: sku.value.trim(),
    name: name.value.trim(),
    price: Number(price.value) || 0,
    stock: Number(stock.value) || 0,
  });
  sku.value = "";
  name.value = "";
  price.value = "";
  stock.value = "";
}
</script>

<template>
  <div
    class="bg-white rounded-xl shadow-sm border border-slate-200 p-6 h-fit"
  >
    <h2 class="text-lg font-semibold text-slate-900 mb-4">
      Create New Product
    </h2>
    <form class="space-y-4" @submit.prevent="handleSubmit">
      <div>
        <label class="block text-xs font-medium text-slate-700 mb-1">
          SKU (Unique Code)
        </label>
        <input
          v-model="sku"
          type="text"
          required
          placeholder="e.g. IPHONE-16-PRO"
          class="w-full text-sm border border-slate-300 rounded-lg px-3 py-2"
        />
      </div>
      <div>
        <label class="block text-xs font-medium text-slate-700 mb-1">
          Product Name
        </label>
        <input
          v-model="name"
          type="text"
          required
          placeholder="e.g. iPhone 16 Pro 256GB"
          class="w-full text-sm border border-slate-300 rounded-lg px-3 py-2"
        />
      </div>
      <div class="grid grid-cols-2 gap-3">
        <div>
          <label class="block text-xs font-medium text-slate-700 mb-1">
            Price
          </label>
          <input
            v-model="price"
            type="number"
            step="0.01"
            min="0"
            required
            placeholder="39900"
            class="w-full text-sm border border-slate-300 rounded-lg px-3 py-2"
          />
        </div>
        <div>
          <label class="block text-xs font-medium text-slate-700 mb-1">
            Initial Stock
          </label>
          <input
            v-model="stock"
            type="number"
            min="0"
            required
            placeholder="50"
            class="w-full text-sm border border-slate-300 rounded-lg px-3 py-2"
          />
        </div>
      </div>
      <button
        type="submit"
        class="w-full bg-emerald-600 hover:bg-emerald-700 text-white text-sm font-medium py-2.5 rounded-lg transition"
      >
        + Add Product
      </button>
    </form>
  </div>
</template>
