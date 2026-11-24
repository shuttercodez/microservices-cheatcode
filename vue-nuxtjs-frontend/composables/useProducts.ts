import { ref, type Ref } from "vue";
import type {
  CreateProductRequest,
  PaginatedProducts,
  Product,
} from "~/types/product";
import { apiRequest } from "~/utils/api-client";

export function useProducts(apiBaseUrl: Ref<string>) {
  const products = ref<Product[]>([]);
  const page = ref(1);
  const limit = ref(10);
  const total = ref(0);
  const totalPages = ref(0);
  const loading = ref(false);

  async function fetchProducts() {
    loading.value = true;
    try {
      const res = await apiRequest<PaginatedProducts>(
        `/api/v1/products?page=${page.value}&limit=${limit.value}`,
        { cache: "no-store" },
        apiBaseUrl.value
      );
      products.value = res.data || [];
      total.value = res.total;
      totalPages.value = res.total_pages;
    } finally {
      loading.value = false;
    }
  }

  async function createProduct(req: CreateProductRequest): Promise<Product> {
    const created = await apiRequest<Product>(
      "/api/v1/products",
      {
        method: "POST",
        body: JSON.stringify(req),
      },
      apiBaseUrl.value
    );
    await fetchProducts();
    return created;
  }

  async function patchStock(id: string, stock: number): Promise<Product> {
    const updated = await apiRequest<Product>(
      `/api/v1/products/${id}/stock`,
      {
        method: "PATCH",
        body: JSON.stringify({ stock }),
      },
      apiBaseUrl.value
    );
    await fetchProducts();
    return updated;
  }

  async function deleteProduct(id: string): Promise<void> {
    await apiRequest<void>(
      `/api/v1/products/${id}`,
      {
        method: "DELETE",
      },
      apiBaseUrl.value
    );
    await fetchProducts();
  }

  return {
    products,
    page,
    limit,
    total,
    totalPages,
    loading,
    fetchProducts,
    createProduct,
    patchStock,
    deleteProduct,
  };
}
