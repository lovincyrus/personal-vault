import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { StdioServerTransport } from "@modelcontextprotocol/sdk/server/stdio.js";
import { z } from "zod";
import { randomBytes } from "crypto";
import type { CallToolResult } from "@modelcontextprotocol/sdk/types.js";

const server = new McpServer({
  name: "shop",
  version: "0.1.0",
});

const PRODUCTS = [
  { id: "tshirt-mcp", name: "So You Can Live Your Best Life Tee", price: "$0.00", sizes: ["XS", "S", "M", "L", "XL", "2XL"] },
  { id: "mug-boss", name: "World's Best Boss Mug", price: "$5.00" },
];

server.registerTool(
  "shop_list_products",
  {
    description: "List all products available in the shop.",
  },
  async (): Promise<CallToolResult> => ({
    content: [{ type: "text", text: JSON.stringify(PRODUCTS, null, 2) }],
  })
);

server.registerTool(
  "shop_place_order",
  {
    description:
      "Place an order. Requires customer name, email, shipping address, and the product ID. Returns an order confirmation.",
    inputSchema: {
      product_id: z.string().describe("Product ID from shop_list_products"),
      size: z.string().optional().describe("Size (for apparel): XS, S, M, L, XL, 2XL"),
      first_name: z.string().describe("Customer first name"),
      last_name: z.string().describe("Customer last name"),
      email: z.string().describe("Customer email"),
      street: z.string().describe("Street address"),
      city: z.string().describe("City"),
      state: z.string().describe("State / province code"),
      zip: z.string().describe("ZIP / postal code"),
      country: z.string().describe("Country code (e.g. US)"),
      payment_last4: z.string().optional().describe("Last 4 digits of payment card"),
      payment_brand: z.string().optional().describe("Card brand (e.g. Visa, Mastercard)"),
    },
  },
  async (args): Promise<CallToolResult> => {
    const product = PRODUCTS.find((p) => p.id === args.product_id);
    if (!product) {
      return {
        content: [{ type: "text", text: `Product "${args.product_id}" not found.` }],
        isError: true,
      };
    }

    const orderId = randomBytes(6).toString("hex");
    const confirmation = {
      order_id: orderId,
      status: "confirmed",
      product: product.name,
      size: args.size ?? "N/A",
      price: product.price,
      customer: {
        name: `${args.first_name} ${args.last_name}`,
        email: args.email,
      },
      shipping: {
        street: args.street,
        city: args.city,
        state: args.state,
        zip: args.zip,
        country: args.country,
      },
      payment: args.payment_last4
        ? { card: `••••${args.payment_last4}`, brand: args.payment_brand ?? "unknown" }
        : undefined,
    };

    return {
      content: [{ type: "text", text: JSON.stringify(confirmation, null, 2) }],
    };
  }
);

async function main() {
  const transport = new StdioServerTransport();
  await server.connect(transport);
}

main().catch((e) => {
  process.stderr.write(`shop-mcp fatal: ${e}\n`);
  process.exit(1);
});
