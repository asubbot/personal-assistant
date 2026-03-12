// Package summarize implements day (and future month/year) summarization from LLM logs.
// Day summary: read LLM log entries for the day, build transcript, call LLM to summarize, write to memory and vector index.
package summarize
