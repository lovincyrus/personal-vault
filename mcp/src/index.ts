#!/usr/bin/env node
import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { StdioServerTransport } from "@modelcontextprotocol/sdk/server/stdio.js";
import { z } from "zod";
import { getStatus, getSchema, getField, listFields, getContext, setField, provisionServiceToken } from "./client.js";
import type { CallToolResult } from "@modelcontextprotocol/sdk/types.js";
import { getScope, scopeAllows, scopeAllowsCategory } from "./scope.js";

function ok(text: string): CallToolResult {
  return { content: [{ type: "text", text }] };
}

function err(message: string): CallToolResult {
  return { content: [{ type: "text", text: message }], isError: true };
}

const scope = getScope();

const server = new McpServer({
  name: "vault",
  version: "0.1.0",
});

server.registerTool(
  "vault_status",
  {
    description:
      "Check if the vault server is running and unlocked. Returns initialization state, lock status, field count, and categories. This is a public endpoint that works without authentication.",
  },
  async (): Promise<CallToolResult> => {
    try {
      const status = await getStatus();
      return ok(JSON.stringify(status, null, 2));
    } catch (e) {
      return err((e as Error).message);
    }
  }
);

server.registerTool(
  "vault_schema",
  {
    description:
      "Retrieve the recommended vault schema with canonical field names, descriptions, and default sensitivity tiers. Use this to discover the correct field IDs before writing to the vault. This is a public endpoint that works without authentication.",
  },
  async (): Promise<CallToolResult> => {
    try {
      const schema = await getSchema();
      return ok(JSON.stringify(schema, null, 2));
    } catch (e) {
      return err((e as Error).message);
    }
  }
);

server.registerTool(
  "vault_get",
  {
    description:
      "Retrieve a single decrypted field from the vault. Returns the field value, category, sensitivity tier, and metadata.",
    inputSchema: {
      id: z
        .string()
        .describe(
          "Field ID in category.field_name format, e.g. identity.full_name"
        ),
    },
  },
  async ({ id }): Promise<CallToolResult> => {
    if (!scopeAllows(scope, id)) {
      return err(`scope denied: ${id} is not in VAULT_SCOPE`);
    }
    try {
      const field = await getField(id);
      return ok(JSON.stringify(field, null, 2));
    } catch (e) {
      return err((e as Error).message);
    }
  }
);

server.registerTool(
  "vault_list",
  {
    description:
      "List all field metadata in the vault (IDs, categories, sensitivity tiers). Does not return decrypted values — use vault_get or vault_context for values.",
  },
  async (): Promise<CallToolResult> => {
    try {
      const fields = await listFields();
      const filtered = fields.filter((f) => scopeAllows(scope, f.id));
      return ok(JSON.stringify(filtered, null, 2));
    } catch (e) {
      return err((e as Error).message);
    }
  }
);

server.registerTool(
  "vault_context",
  {
    description:
      "Retrieve all decrypted fields grouped by category. Returns the full personal context bundle. Note: this returns ALL fields including sensitive and critical tiers. All data passes through the LLM provider's inference pipeline.",
  },
  async (): Promise<CallToolResult> => {
    try {
      const ctx = await getContext();
      for (const cat of Object.keys(ctx.categories)) {
        if (!scopeAllowsCategory(scope, cat)) {
          delete ctx.categories[cat];
          continue;
        }
        ctx.categories[cat] = ctx.categories[cat].filter((f) =>
          scopeAllows(scope, f.id)
        );
        if (ctx.categories[cat].length === 0) {
          delete ctx.categories[cat];
        }
      }
      return ok(JSON.stringify(ctx, null, 2));
    } catch (e) {
      return err((e as Error).message);
    }
  }
);

server.registerTool(
  "vault_set",
  {
    description:
      "Save a field to the user's encrypted vault. Creates or updates a field. Use this when the user provides personal information that isn't already in the vault (e.g. t-shirt size, dietary preferences). Always ask the user for confirmation before saving: 'Want me to save M as your t-shirt size for next time?' Call vault_schema first to discover recommended field names.",
    inputSchema: {
      id: z
        .string()
        .describe(
          "Field ID in category.field_name format, e.g. identity.tshirt_size"
        ),
      value: z.string().describe("The value to encrypt and store"),
      sensitivity: z
        .enum(["public", "standard", "sensitive", "critical"])
        .optional()
        .describe("Sensitivity tier (default: standard)"),
    },
  },
  async ({ id, value, sensitivity }): Promise<CallToolResult> => {
    if (!scopeAllows(scope, id)) {
      return err(`scope denied: ${id} is not in VAULT_SCOPE`);
    }
    try {
      await setField(id, value, sensitivity);
      return ok(`Saved ${id}`);
    } catch (e) {
      return err((e as Error).message);
    }
  }
);

async function main() {
  // When scoped, provision a service token so the audit log reflects the scope
  if (scope !== "*") {
    const consumer = process.env.VAULT_CONSUMER ?? "mcp";
    await provisionServiceToken(consumer, scope);
  }

  const transport = new StdioServerTransport();
  await server.connect(transport);
}

main().catch((e) => {
  process.stderr.write(`vault-mcp fatal: ${e}\n`);
  process.exit(1);
});
