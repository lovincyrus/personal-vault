# Shopping Demo — MCP Integration Test

This is a development demo for testing MCP server integration. You have two MCP servers registered that you should use to test the end-to-end flow:

- **vault** — MCP server exposing encrypted personal context (name, email, address) with read and write capability
- **shop** — MCP server exposing a mock product catalog and order API

This is NOT real shopping. The shop MCP is a mock server running locally that returns fake order confirmations. Your job is to exercise these MCP tools to demonstrate the integration.

## How to run the demo

When the user asks to "buy" something or "test the demo":

1. Call `vault_status` to verify the vault is running and unlocked.
2. Call `vault_context` to retrieve the user's identity, address, and payment fields.
3. Call `shop_list_products` to list available mock products.
4. If any information is missing (e.g. size for apparel), ask the user.
5. If payment card info exists in the vault, show the user what's on file: "Paying with Visa ending in ••1234". If no card is on file, ask the user for their card details (number, expiry, cardholder name, brand).
6. Call `shop_place_order` with the product ID, vault data mapped to order fields, and `payment_last4` / `payment_brand` extracted from the stored card. **Never pass the full card number to the shop API.**
7. Present the mock order confirmation.
8. **After the order**, if the user provided any new personal information (like t-shirt size or card details), offer to save it to the vault for next time using `vault_set`. Example: "Want me to save your Visa ending in 1234 to the vault so it's ready next time?"

## Learning new fields

When the user provides personal information that isn't in the vault yet, always offer to save it:

- Ask for confirmation before calling `vault_set`
- Use a sensible field ID (e.g. `identity.tshirt_size`, `preferences.color`)
- Set sensitivity to `public` for preferences, `standard` for personal details
- **Payment card fields must always use `critical` sensitivity tier**

### Payment card fields

Store card details across these vault fields:

| Vault field | Example | Sensitivity |
|---|---|---|
| `payment.card_number` | `4111111111111234` | `critical` |
| `payment.card_expiry` | `12/28` | `critical` |
| `payment.cardholder_name` | `Cool Cucumber` | `critical` |
| `payment.card_brand` | `Visa` | `standard` |

When displaying stored card info, always mask it: show only the brand and last 4 digits (e.g. "Visa ending in ••1234"). Never display the full card number back to the user.

## Field mapping

| Vault field | Shop order field |
|---|---|
| `identity.first_name` | `first_name` |
| `identity.last_name` | `last_name` |
| `identity.email` | `email` |
| `addresses.home_street` | `street` |
| `addresses.home_city` | `city` |
| `addresses.home_state` | `state` |
| `addresses.home_zip` | `zip` |
| `addresses.home_country` | `country` |
| `payment.card_number` (last 4 only) | `payment_last4` |
| `payment.card_brand` | `payment_brand` |
