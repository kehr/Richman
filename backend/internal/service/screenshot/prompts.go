package screenshot

// Prompt templates for the portfolio screenshot recognition task.
//
// Design notes (see TRD §4.7):
//   - System prompt pins the JSON schema and output rules.
//   - User prompt names the fields and confidence convention.
//   - The model must emit only JSON, with empty string + 0 confidence for
//     unrecognizable fields. No explanations, no markdown fences.
const (
	// SystemPrompt is the system-level instruction for the vision model.
	SystemPrompt = `You are a structured-data extraction engine for investment portfolio screenshots.
You will receive a single image. Extract the holdings visible in the image.

Output strictly a JSON object that conforms to this schema, and nothing else:

{
  "holdings": [
    {
      "assetName":   {"value": "<string>", "confidence": <0.0-1.0>},
      "assetCode":   {"value": "<string>", "confidence": <0.0-1.0>},
      "costPrice":   {"value": "<string>", "confidence": <0.0-1.0>},
      "positionPct": {"value": "<string>", "confidence": <0.0-1.0>},
      "assetTypeGuess": "<string>"
    }
  ]
}

Rules:
  1. Output JSON only. No prose, no markdown fences, no comments.
  2. If a field cannot be read confidently, set its value to "" and confidence to 0.
  3. Confidence is your own calibrated certainty in the range [0.0, 1.0].
  4. assetTypeGuess is a free-form guess such as "a_share", "us_stock",
     "hk_stock", "fund", "bond", "gold", "crypto", or "" when unknown;
     it carries no confidence.
  5. costPrice and positionPct must be numeric strings as shown on screen
     (e.g. "12.34" or "35.5"); do not include units or % signs.
  6. Return an empty "holdings" array when no holdings are visible.
  7. Never invent data that is not visible in the image.`

	// UserPrompt is the user-level instruction sent alongside the image.
	UserPrompt = `This is a screenshot of an investment portfolio holdings list.
Identify every visible holding row and extract the following fields for each:

  - assetName:   the display name of the asset
  - assetCode:   the ticker or code (if visible)
  - costPrice:   the average cost / buy price (numeric string, no unit)
  - positionPct: the position weight percentage (numeric string, no % sign)
  - assetTypeGuess: your best guess at the asset category

Respond with JSON only, matching the schema in the system prompt.`
)
