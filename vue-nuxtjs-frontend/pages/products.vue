<script setup lang="ts">
import { onMounted, watch } from "vue";
import type { CreateProductRequest, Product } from "~/types/product";

const router = useRouter();
const { apiBaseUrl, isAuthenticated, showSuccess, showError } = useConsole();
const productsState = useProducts(apiBaseUrl);

async function loadProducts() {
  if (!isAuthenticated.value) return;
  try {
    await productsState.fetchProducts();
  } catch (err: unknown) {
    showError(err instanceof Error ? err.message : "Failed to load products");
  }
}

onMounted(() => {
  loadProducts();
});

watch([productsState.page, apiBaseUrl], () => {
  loadProducts();
});

async function handleCreateProduct(req: CreateProductRequest) {
  try {
    const created = await productsState.createProduct(req);
    showSuccess(`Created product "${created.name}" (${created.sku})`);
  } catch (err: unknown) {
    showError(err instanceof Error ? err.message : "Failed to create product");
  }
}

async function handlePatchStock(productId: string, newStock: number) {
  try {
    const updated = await productsState.patchStock(productId, newStock);
    showSuccess(
      `Updated stock for ${updated.sku} to ${updated.stock} (v${updated.version})`
    );
  } catch (err: unknown) {
    showError(err instanceof Error ? err.message : "Failed to update stock");
  }
}

async function handleDeleteProduct(product: Product) {
  try {
    await productsState.deleteProduct(product.id);
    showSuccess(`Soft-deleted product ${product.sku}`);
  } catch (err: unknown) {
    showError(err instanceof Error ? err.message : "Failed to delete product");
  }
}

function handleSelectForOrder(productId: string) {
  router.push({ path: "/orders", query: { productId } });
}
</script>

<template>
  <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
    <ProductsProductForm @create="handleCreateProduct" />
    <ProductsProductTable
      :products="productsState.products.value"
      :page="productsState.page.value"
      :total-pages="productsState.totalPages.value"
      :total="productsState.total.value"
      @page-change="productsState.page.value = $event"
      @patch-stock="handlePatchStock"
      @delete="handleDeleteProduct"
      @select-for-order="handleSelectForOrder"
    />
  </div>
</template>
