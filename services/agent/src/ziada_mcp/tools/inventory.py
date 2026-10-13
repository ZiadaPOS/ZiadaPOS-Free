"""Inventory tools: product categories, products and stock management."""
from __future__ import annotations

from typing import Any

from ._base import param


def register(mcp: Any, client: Any) -> None:
    @mcp.tool()
    async def list_categories() -> list[dict]:
        """List the shop's product categories (organisation-scoped, shared by all stores)."""
        return await client.request("GET", "/inventory/categories")

    @mcp.tool()
    async def create_category(name: str, description: str = "") -> dict:
        """Create a product category. Args: name (required), description."""
        return await client.request("POST", "/inventory/categories",
                                    json={"name": name, "description": description})

    @mcp.tool()
    async def update_category(category_id: str, name: str | None = None,
                              description: str | None = None,
                              sort_order: int | None = None,
                              is_active: bool | None = None) -> dict:
        """Update a product category. Only provided fields are changed."""
        body: dict[str, Any] = {}
        if name is not None:
            body["name"] = name
        if description is not None:
            body["description"] = description
        if sort_order is not None:
            body["sort_order"] = sort_order
        if is_active is not None:
            body["is_active"] = is_active
        return await client.request("PATCH", f"/inventory/categories/{category_id}", json=body)

    @mcp.tool()
    async def delete_category(category_id: str) -> str:
        """Delete a product category. Use with care — products keep their data but lose the category link."""
        await client.request("DELETE", f"/inventory/categories/{category_id}")
        return "Category deleted"

    # ── Products ────────────────────────────────────────────────────────────

    @mcp.tool()
    async def list_products(search: str | None = None,
                            category_id: str | None = None,
                            status: str | None = None,
                            low_stock: bool | None = None,
                            page: int = 1,
                            page_size: int = 20) -> dict:
        """List the store's products with filters. status: active/inactive; low_stock: only items at/below reorder point."""
        return await client.request("GET", "/inventory/products", params=param(
            search=search, category_id=category_id, status=status,
            low_stock=low_stock, page=page, page_size=page_size))

    @mcp.tool()
    async def create_product(
        name: str,
        selling_price: int,
        cost_price: int = 0,
        category_id: str | None = None,
        sku: str | None = None,
        barcode: str | None = None,
        stock: int = 0,
        reorder_point: int = 5,
        image: str | None = None,
        tax_rate: float = 18,
        description: str | None = None,
        is_service: bool = False,
        is_active: bool = True,
    ) -> dict:
        """Create a product. All prices are in TZS (whole shillings). stock starts the opening quantity."""
        return await client.request("POST", "/inventory/products", json={
            "name": name, "selling_price": selling_price, "cost_price": cost_price,
            "category_id": category_id or None, "sku": sku or None, "barcode": barcode or None,
            "stock": stock, "reorder_point": reorder_point, "image": image or None,
            "tax_rate": tax_rate, "description": description or None,
            "is_service": is_service, "is_active": is_active,
        })

    @mcp.tool()
    async def get_product(product_id: str) -> dict:
        """Get one product by its UUID."""
        return await client.request("GET", f"/inventory/products/{product_id}")

    @mcp.tool()
    async def get_product_by_barcode(barcode: str) -> dict:
        """Find a product by its barcode (scanner code)."""
        return await client.request("GET", f"/inventory/products/barcode/{barcode}")

    @mcp.tool()
    async def update_product(
        product_id: str,
        name: str | None = None,
        selling_price: int | None = None,
        cost_price: int | None = None,
        category_id: str | None = None,
        sku: str | None = None,
        barcode: str | None = None,
        reorder_point: int | None = None,
        image: str | None = None,
        tax_rate: float | None = None,
        description: str | None = None,
        is_service: bool | None = None,
        is_active: bool | None = None,
    ) -> dict:
        """Update a product. Only provided fields are changed. stock is NOT editable here — use adjust_stock."""
        body: dict[str, Any] = {}
        for k, v in {
            "name": name, "selling_price": selling_price, "cost_price": cost_price,
            "category_id": category_id, "sku": sku, "barcode": barcode,
            "reorder_point": reorder_point, "image": image, "tax_rate": tax_rate,
            "description": description, "is_service": is_service, "is_active": is_active,
        }.items():
            if v is not None:
                body[k] = v
        return await client.request("PATCH", f"/inventory/products/{product_id}", json=body)

    @mcp.tool()
    async def delete_product(product_id: str) -> str:
        """Deactivate (soft-delete) a product. It stays in history but can't be sold."""
        await client.request("DELETE", f"/inventory/products/{product_id}")
        return "Product deleted"

    # ── Stock ───────────────────────────────────────────────────────────────

    @mcp.tool()
    async def list_low_stock() -> list[dict]:
        """List products at or below their reorder point (need restocking)."""
        return await client.request("GET", "/inventory/low-stock")

    @mcp.tool()
    async def adjust_stock(product_id: str, qty_change: int, reason: str,
                           note: str | None = None) -> dict:
        """Adjust stock level of a product. qty_change is POSITIVE to add stock (restock) and NEGATIVE to remove (damage/loss/shrinkage). reason: e.g. 'restock', 'damage', 'correction', 'return'."""
        return await client.request("POST", f"/inventory/products/{product_id}/adjust", json={
            "qty_change": qty_change, "reason": reason, "note": note or "",
        })

    @mcp.tool()
    async def list_stock_adjustments(product_id: str, limit: int = 50) -> list[dict]:
        """List the stock-adjustment history for one product."""
        return await client.request("GET", f"/inventory/products/{product_id}/adjustments",
                                    params=param(limit=limit))
