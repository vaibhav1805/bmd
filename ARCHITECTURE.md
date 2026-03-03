# BMD Architecture

Technical deep-dive into BMD's design, components, and features.

## Core Components

### Rendering Engine
**Goal:** Render all markdown elements beautifully in the terminal

- **Parser:** Goldmark wrapper for AST generation
- **Renderer:** ANSI terminal renderer with 256-color palette
- **Syntax Highlighting:** 20+ programming languages via Chroma
- **Elements:** Full support for headings, lists, tables, code blocks, blockquotes, images, links

**Files:** `internal/parser/`, `internal/renderer/`

### Terminal UI Framework
**Goal:** Event-driven user interface for interactive editing and browsing

- **Engine:** Bubbletea (Go TUI framework)
- **Input Handling:** Keyboard and mouse event processing
- **Rendering:** Double-buffered output with ANSI escape codes
- **Themes:** 5 built-in color themes (Default, Ocean, Forest, Sunset, Midnight)
- **Modes:** View, Edit, Search, Directory Browse, Graph

**Files:** `internal/tui/`, `internal/terminal/`

### Navigation & Link Following
**Goal:** Enable users to move between files and understand relationships

- **Link Registry:** Maps terminal positions to URLs
- **Path Resolution:** Relative and absolute path handling
- **History Stack:** Back/forward navigation with cursor preservation
- **Link Detection:** Extracts markdown links from AST

**Files:** `internal/nav/`, `internal/tui/linkreg.go`

### Search System
**Goal:** Find content within and across documents

- **In-Document Search:** Pattern matching with term highlighting
- **Full-Text Search:** BM25 ranking algorithm for relevance
- **Cross-Document:** Search all markdown files in a directory
- **Results:** Highlighted matches with context snippets

**Files:** `internal/search/`, `internal/knowledge/`

### Knowledge Graph & Agent Interface
**Goal:** Enable programmatic queries and architectural analysis

- **Full-Text Indexing:** BM25 index with configurable tokenization
- **Knowledge Graph:** Document relationships and dependency detection
- **Component Detection:** Identifies components from documentation structure
- **Dependency Analysis:** Transitive dependency chains, cycles, depth analysis
- **Persistence:** SQLite database with WAL mode for concurrent access
- **CLI Interface:** Programmatic query commands (query, depends, components, graph)

**Files:** `internal/knowledge/`, search/BM25, graph detection

### Editor
**Goal:** Edit markdown files with syntax highlighting and persistence

- **Text Buffer:** Efficient line-based editing with vim-like cursor movement
- **Syntax Highlighting:** Pattern-based markdown highlighting with ANSI colors
- **File Persistence:** Atomic write pattern (temp file + rename)
- **Undo/Redo:** Full edit history with snapshot restoration
- **Navigation:** Jump to line, Page Up/Down, Ctrl+Home/End

**Files:** `internal/editor/`, edit mode in `internal/tui/viewer.go`

### Directory Browser
**Goal:** Interactive file listing and navigation in the terminal

- **Directory Scanning:** Recursive .md file discovery with metadata
- **File Listing:** Sortable view with line count, size, modification time
- **Navigation:** Keyboard-driven with saved cursor position
- **Split-Pane Mode:** Dual-pane layout with file list and preview (Beta)
- **File Preview:** Real-time markdown preview with full syntax highlighting
- **Cross-File Search:** BM25 search results with context snippets

**Files:** `internal/tui/viewer.go` (directory state and rendering)

### MCP Server
**Goal:** Expose all knowledge tools as native MCP endpoints for persistent agent integration

- **Protocol:** Model Context Protocol (MCP) via stdin/stdout
- **SDK:** mark3labs/mcp-go (community MCP SDK for Go)
- **Tools:** bmd/query, bmd/index, bmd/depends, bmd/components, bmd/graph, bmd/context, bmd/graph_crawl
- **Zero subprocess overhead:** Single process handles all agent requests
- **CONTRACT-01 compliance:** All responses wrapped in JSON envelope (status/code/message/data)
- **Startup:** `bmd serve --mcp` — blocks until process is killed

```
Agent (MCP client)
    ↓ stdin (JSON-RPC)
bmd serve --mcp (MCP server)
    ↓ delegates to knowledge.*Cmd functions
SQLite index + Knowledge graph
    ↑ stdout (JSON-RPC response)
Agent (receives result)
```

**Files:** `internal/mcp/server.go`, `internal/mcp/handlers.go`

### Image Rendering
**Goal:** Display images in compatible terminal emulators

- **Protocol Detection:** Auto-detects Kitty, iTerm2, Sixel, unicode
- **Supported Terminals:** Alacritty, Kitty, iTerm2, WezTerm, xterm, others
- **Fallback:** Unicode blocks and alt text when protocol unavailable
- **Performance:** Minimal overhead, no external dependencies

**Files:** `internal/renderer/images.go`

## Pipeline Flows

### View/Edit Pipeline
```
Markdown File
    ↓
Goldmark Parser (AST)
    ↓
Internal Renderer (ANSI codes)
    ↓
Terminal UI (Bubbletea)
    ↓
Rendered Output
```

### Search Pipeline
```
Query
    ↓
Pattern Matcher / BM25 Index
    ↓
Highlighted Results
    ↓
Terminal Display
```

### Knowledge System Pipeline
```
Markdown Directory
    ↓
File Scanner (find all .md)
    ↓
BM25 Indexing (full-text)
    ↓
Graph Builder (explicit markdown links)
    ↓
DiscoverRelationships (co-occurrence, structural, semantic)
    ↓
Merge discovered edges into graph
    ↓
Component Detection (microservices)
    ↓
SQLite Persistence
    ↓
Relationship Manifest (.bmd-relationships-discovered.yaml)
    ↓
User Review & Optional LLM Validation
    ↓
Accepted Relationships (.bmd-relationships.yaml)
    ↓
CLI Query Interface
```

### Graph Traversal Algorithm

The `crawl` command performs multi-start BFS traversal of the knowledge graph, designed for agents to explore dependency chains and assess impact.

**Algorithm: Multi-Start BFS**
```
1. Enqueue all valid start nodes at depth 0
2. For each node in queue:
   a. Skip if depth exceeds MaxDepth
   b. Collect neighbors based on Direction:
      - "forward": outgoing edges (BySource map)
      - "backward": incoming edges (ByTarget map)
      - "both": union of both
   c. For each neighbor:
      - If unvisited: record depth, parent, enqueue
      - If visited: append additional parent (multi-path tracking)
3. Post-traversal: populate EdgesOut for all discovered nodes
4. If IncludeCycles: run DFS cycle detection on discovered subgraph
```

**Cycle Detection (Post-BFS DFS)**

Uses three-color marking (white/gray/black) on the discovered subgraph. Back-edges to gray nodes indicate cycles. Cycles are classified as "direct" (same branch ancestry) or "cross_branch" (nodes from different start branches). Deduplication uses canonical rotation of cycle paths.

**Multi-Parent Tracking**

When a node is reachable via multiple paths (e.g., diamond dependency pattern A->B->D, A->C->D), all discovering parents are recorded in `NodeInfo.Parents`. The shortest BFS depth is preserved (first visit wins).

**Performance:** <100ms for 50-node graphs, <1ms for typical crawls. The BFS is O(V+E) where V=nodes discovered, E=edges traversed.

**Files:** `internal/knowledge/crawl.go` (engine), `internal/knowledge/output.go` (formatters)

## Knowledge Architecture Deep Dive

### BM25 Full-Text Indexing Architecture

The BM25 index is built during `bmd index` and enables fast keyword-based search across all documents.

```
Markdown Files
    │
    ├── auth-service.md
    │   Content: "OAuth2, JWT, token validation"
    │
    ├── api-gateway.md
    │   Content: "request routing, middleware, authentication"
    │
    └── user-service.md
        Content: "user profiles, roles, permissions"

    ↓ (ScanDirectory)

Document Collection
    │
    ├── auth-service.md (ID: auth-service.md)
    │   RelPath: "services/auth-service.md"
    │   Title: "OAuth2 Service"
    │   Content: "OAuth2, JWT, token validation..."
    │
    ├── api-gateway.md (ID: api-gateway.md)
    │   RelPath: "services/api-gateway.md"
    │   Title: "API Gateway"
    │   Content: "request routing, middleware..."
    │
    └── user-service.md (ID: user-service.md)
        RelPath: "services/user-service.md"
        Title: "User Service"
        Content: "user profiles, roles..."

    ↓ (Tokenization: lowercase, stop words removed, stemming)

Term Index (BM25 Postings)
    │
    ├── "oauth" → [auth-service.md (freq: 3, TF-IDF: 2.4)]
    ├── "jwt" → [auth-service.md (freq: 2, TF-IDF: 1.8)]
    ├── "token" → [auth-service.md (freq: 4, TF-IDF: 3.1), api-gateway.md (freq: 1, TF-IDF: 0.9)]
    ├── "routing" → [api-gateway.md (freq: 2, TF-IDF: 1.7)]
    ├── "middleware" → [api-gateway.md (freq: 1, TF-IDF: 0.8)]
    ├── "user" → [user-service.md (freq: 5, TF-IDF: 4.2)]
    └── "permission" → [user-service.md (freq: 1, TF-IDF: 0.7)]

    ↓ (Query: "token validation")

BM25 Ranking
    │
    ├── [1] auth-service.md (score: 5.2)  ← "token" (3.1) + "validation" (2.1)
    ├── [2] api-gateway.md (score: 0.9)   ← "token" only (0.9)
    └── [3] user-service.md (score: 0)    ← no match

    ↓ (SQLite Persistence)

Search Index Database
    bm25_documents
    ├── doc_id | file_path | title | hash
    ├── 1 | services/auth-service.md | OAuth2 Service | abc123
    ├── 2 | services/api-gateway.md | API Gateway | def456
    └── 3 | services/user-service.md | User Service | ghi789

    bm25_stats (IDF cache, BM25 parameters)
    index_entries (term postings, TF-IDF scores)
```

### PageIndex Semantic Search Architecture

PageIndex adds LLM-powered semantic search on top of BM25, enabling intent-based queries with reasoning traces.

```
Markdown Document: auth-service.md
┌──────────────────────────────────────────────┐
│ # OAuth2 Service                             │
│                                              │
│ Handles token validation and JWT signing.   │
│                                              │
│ ## Architecture                              │
│ - Validates JWT tokens                      │
│ - Caches tokens in redis                    │
│ - Exposes /auth/validate endpoint           │
│                                              │
│ ## Usage Examples                            │
│ - POST /auth/validate with JWT              │
│ - Returns 401 for invalid tokens            │
└──────────────────────────────────────────────┘
    ↓ (bmd index --strategy pageindex)
    ↓ (Hierarchical tree construction)

PageIndex Tree File (.bmd-tree.json)
    │
    └── FileTree
        ├── FilePath: "auth-service.md"
        ├── Title: "OAuth2 Service"
        ├── Summary: "Service for OAuth2 token validation, JWT signing..."
        │
        └── Children (Sections)
            ├── TreeNode
            │   ├── Content: "Handles token validation and JWT signing."
            │   ├── Heading: "# OAuth2 Service"
            │   ├── Summary: "Core service that validates OAuth2 tokens..."
            │   └── Children
            │       ├── TreeNode
            │       │   ├── Content: "- Validates JWT tokens..."
            │       │   ├── Heading: "## Architecture"
            │       │   ├── Summary: "Architectural components: JWT validation..."
            │       │   └── Children: [...]
            │       │
            │       └── TreeNode
            │           ├── Content: "- POST /auth/validate with JWT..."
            │           ├── Heading: "## Usage Examples"
            │           ├── Summary: "How to call the OAuth2 service API..."
            │           └── Children: [...]
            │
            └── TreeNode
                ├── Content: "Returns 401 for invalid tokens"
                ├── Heading: "## Error Handling"
                └── Summary: "Error responses and status codes..."

    ↓ (Run subprocess: pageindex index --file auth-service.md)

PageIndex Indexing Process (Subprocess)
    │
    └── pageindex index
        ├── Input: Tree JSON (headings, content, summaries)
        ├── Model: "claude-opus" (default LLM)
        ├── Embeddings: Generate semantic vectors for each section
        │   └── [0.234, 0.891, 0.124, ...] ← embedding vector
        │
        └── Output: Indexed tree with embeddings

    ↓ (Save tree file locally)

Stored Tree Files
    ├── docs/auth-service.bmd-tree.json
    ├── docs/api-gateway.bmd-tree.json
    └── docs/user-service.bmd-tree.json

Query Processing with PageIndex:

    Query: "How do we validate user tokens?"

    ↓ (bmd query "How do we validate tokens?" --strategy pageindex)

    Step 1: Load all .bmd-tree.json files from indexed directory

    Step 2: Generate query embedding
        pageindex query --query "How do we validate user tokens?"
        ├── Input: Natural language question
        ├── Model: "claude-opus"
        └── Output: Query embedding [0.156, 0.923, 0.087, ...]

    Step 3: Semantic similarity search
        For each tree section:
        ├── Compute cosine similarity(query_embedding, section_embedding)
        │   auth-service.md § Architecture: 0.87 ✓ HIGH MATCH
        │   auth-service.md § Usage: 0.79 ✓ MATCH
        │   api-gateway.md § Error Handling: 0.45 ✗ low match
        │   user-service.md § Permissions: 0.31 ✗ very low match
        │
        └── Rank by similarity score

    Step 4: Return top results with reasoning trace
        [1] auth-service.md § Architecture (score: 0.87)
            "Covers JWT token validation, the core mechanism..."
            Content: "- Validates JWT tokens
                      - Checks signature and expiry..."

        [2] auth-service.md § Usage (score: 0.79)
            "Explains how to call token validation API..."
            Content: "- POST /auth/validate with JWT
                      - Returns 200 with validated token..."

Comparison: BM25 vs PageIndex

    BM25 Search (Keyword-based):
    ├── Query: "token validation"
    ├── Matching: Exact terms "token" + "validation"
    ├── Results:
    │   [1] auth-service.md (contains both terms)
    │   [2] api-gateway.md (contains "token")
    │   [3] user-service.md (contains "validation")
    ├── Speed: <8ms
    └── Cost: 0 (no LLM)

    PageIndex Search (Semantic):
    ├── Query: "How do we validate user tokens?"
    ├── Matching: Intent-based (token validation concept)
    ├── Results:
    │   [1] auth-service.md § Architecture (0.87 similarity)
    │   [2] auth-service.md § Usage (0.79 similarity)
    │   [3] api-gateway.md § Request Flow (0.62 similarity)
    ├── Speed: ~200ms (includes LLM)
    └── Cost: LLM API calls per query

Strategy Selection (Command-line):

    # Fast keyword search (no dependencies)
    bmd query "token validation" --strategy bm25 --dir ./docs

    # Semantic search with reasoning (requires pageindex)
    bmd query "How do we validate tokens?" --strategy pageindex --dir ./docs

    # Auto-detection (tries pageindex, falls back to BM25)
    bmd query "token validation" --dir ./docs

Fallback Behavior:

    User: bmd query "validate tokens" --strategy pageindex --dir ./docs

    ↓

    Check: Are .bmd-tree.json files present?
    ├── YES → Use PageIndex semantic search
    │
    └── NO → Fall back to BM25
                ├── Warn: "PageIndex trees not found, using BM25"
                ├── Command: "Run 'bmd index --strategy pageindex' first"
                └── Return: BM25 results instead

    Check: Is pageindex binary installed?
    ├── YES → Use it
    │
    └── NO → Fall back to BM25
                ├── Error: "pageindex binary not found"
                ├── Suggestion: "pip install pageindex"
                └── Return: BM25 results with warning

Integration with Knowledge System:

    Graph Building:
    ├── BM25 index ✓ (always)
    └── Knowledge graph ✓ (always)

    + PageIndex Trees:
    ├── Optional .bmd-tree.json files
    ├── Parallel to BM25 (no interference)
    └── Used only when --strategy pageindex requested

    + Context Assembly:
    ├── bmd context "question" --dir ./docs
    ├── Uses PageIndex if trees exist
    ├── Falls back to BM25 if not
    └── Returns assembled context blocks (§ notation)

Files and Dependencies:

    Core (always present):
    ├── .bmd/knowledge.db (SQLite: BM25 + graph)
    └── internal/knowledge/search.go (BM25 implementation)

    PageIndex (optional):
    ├── ~/.local/bin/pageindex (subprocess binary)
    ├── .bmd-tree.json files (one per markdown)
    └── internal/knowledge/pageindex.go (integration)

Environment Variables:

    # Default strategy for all commands
    export BMD_STRATEGY=pageindex

    # pageindex subprocess path (auto-detected if in PATH)
    export BMD_PAGEINDEX_BIN=/usr/local/bin/pageindex

    # Model for embedding generation
    export BMD_PAGEINDEX_MODEL=claude-opus
```

### Knowledge Graph Construction

The knowledge graph is built by extracting relationships from markdown content (links, code mentions, service references).

```
Markdown Document: auth-service.md
┌─────────────────────────────────────────┐
│ # OAuth2 Service                        │
│                                         │
│ Handles token validation via [[api..]]  │
│                                         │
│ Depends on:                             │
│ - jwt validation library                │
│ - redis (caching)                       │
│                                         │
│ See also: [[user-service.md]]          │
└─────────────────────────────────────────┘
    ↓ (Link Extraction: [[...]])
    ↓ (Code Extraction: backtick spans)
    ↓ (Mention Detection: service names)

Extracted Edges (Relationships)
    │
    ├── auth-service.md → api-gateway.md   [ConfidenceLink: 1.0, "handles token validation"]
    ├── auth-service.md → user-service.md  [ConfidenceLink: 1.0, "see also reference"]
    ├── auth-service.md → redis            [ConfidenceMention: 0.7, "caching library"]
    └── auth-service.md → jwt              [ConfidenceMention: 0.7, "validation library"]

    ↓ (All documents processed)

Knowledge Graph Structure
    │
    Nodes (Documents):
    ├── auth-service.md    [Type: document, Title: "OAuth2 Service"]
    ├── api-gateway.md     [Type: document, Title: "API Gateway"]
    ├── user-service.md    [Type: document, Title: "User Service"]
    ├── redis              [Type: component, Title: "Redis Cache"]
    └── jwt                [Type: library, Title: "JWT Library"]

    Edges (Relationships):
    ├── ID: "auth-api"
    │   Source: auth-service.md
    │   Target: api-gateway.md
    │   Type: dependency
    │   Confidence: 1.0 (explicit link)
    │   Evidence: "handles token validation via [[api-gateway.md]]"
    │
    ├── ID: "auth-user"
    │   Source: auth-service.md
    │   Target: user-service.md
    │   Type: dependency
    │   Confidence: 1.0
    │   Evidence: "See also: [[user-service.md]]"
    │
    └── ID: "auth-redis"
        Source: auth-service.md
        Target: redis
        Type: dependency
        Confidence: 0.7 (mention in code)
        Evidence: "redis (caching)"

    ↓ (SQLite Persistence)

Graph Database Tables

    graph_nodes
    ├── id | type | file | title
    ├── auth-service.md | document | services/auth-service.md | OAuth2 Service
    ├── api-gateway.md | document | services/api-gateway.md | API Gateway
    ├── user-service.md | document | services/user-service.md | User Service
    ├── redis | component | NULL | Redis Cache
    └── jwt | library | NULL | JWT Library

    graph_edges
    ├── id | source_id | target_id | type | confidence | evidence
    ├── auth-api | auth-service.md | api-gateway.md | dependency | 1.0 | handles token...
    ├── auth-user | auth-service.md | user-service.md | dependency | 1.0 | see also...
    └── auth-redis | auth-service.md | redis | dependency | 0.7 | redis (caching)
```

### Relationship Discovery Layer

In addition to explicit markdown links, BMD automatically discovers implicit relationships using pattern-matching algorithms:

**Discovery Methods:**
1. **Co-Occurrence Analysis** (0.5–0.75 confidence)
   - Detects service names mentioned in the same document or nearby sections
   - Example: If `auth.md` mentions "redis" and "cache", adds edge auth.md → redis

2. **Structural Pattern Matching** (0.6–0.85 confidence)
   - Analyzes function calls, class references, imports, and configuration references
   - Example: `database.Connect()` or `import { UserService }` trigger edges

3. **Semantic Similarity (TF-IDF)** (0.45–0.75 confidence)
   - Computes term-frequency similarity between documents
   - Example: Documents with similar vocabulary get lower-confidence edges

4. **Named Entity Recognition + SVO Patterns** (0.5–0.8 confidence)
   - Identifies service names (NER) and subject-verb-object patterns
   - Example: "auth-service calls user-service" → creates directed edge

**Discovery Flow:**
```
Indexed Documents
    ↓
Run all 4 discovery algorithms in parallel
    ↓
Aggregate signals (deduplicate by source→target)
    ↓
Generate confidence scores (0.45–0.85 range)
    ↓
Merge into graph structure
    ↓
Create .bmd-relationships-discovered.yaml manifest
    ↓
User review + optional LLM validation
    ↓
User accepts/rejects, saves to .bmd-relationships.yaml
    ↓
Accepted relationships merged into final graph
```

**Confidence Ranges:**
- 1.0: Explicit markdown links (highest confidence)
- 0.8–0.95: Strong structural patterns (imports, function calls)
- 0.65–0.8: Semantic patterns + strong co-occurrence
- 0.45–0.65: Weak co-occurrence, NER-based patterns
- 0.0–0.45: Rejected by user or below threshold

### Graph Traversal: Multi-Start BFS Example

Example: Crawling dependencies starting from `auth-service.md`

```
Starting State:
    FromFiles: ["auth-service.md"]
    Direction: "backward" (who depends on auth-service?)
    MaxDepth: 2

Knowledge Graph:
    ┌─────────────────────────────────────────┐
    │                                         │
    │  api-gateway.md ──depends──> auth-service.md
    │          │                         ▲
    │          │ depends                 │
    │          ▼                         │
    │  user-service.md ──depends────────┘
    │
    └─────────────────────────────────────────┘

BFS Traversal (Backward Direction):

    Queue: [(auth-service.md, depth=0)]
    Visited: {}

    Step 1: Process auth-service.md
    ├── Neighbors (BACKWARD): Incoming edges
    │   ├── api-gateway.md → auth-service.md
    │   └── user-service.md → auth-service.md
    │
    ├── Enqueue api-gateway.md (depth=1, parent=auth-service.md)
    ├── Enqueue user-service.md (depth=1, parent=auth-service.md)
    │
    └── Visited: {auth-service.md}
        Queue: [(api-gateway.md, 1), (user-service.md, 1)]

    Step 2: Process api-gateway.md
    ├── Neighbors (BACKWARD): [no incoming edges in this direction]
    │
    └── Visited: {auth-service.md, api-gateway.md}
        Queue: [(user-service.md, 1)]

    Step 3: Process user-service.md
    ├── Neighbors (BACKWARD): [no new incoming edges]
    │
    └── Visited: {auth-service.md, api-gateway.md, user-service.md}
        Queue: []

    BFS Complete at depth=1

Traversal Result:
    ┌────────────────────────────────────────┐
    │ Discovered Subgraph (depth ≤ 2)       │
    │                                        │
    │ Start:                                 │
    │   auth-service.md ────────────┐        │
    │                               │        │
    │ Depth 1:                      ▼        │
    │   ├── api-gateway.md (parent: auth)   │
    │   └── user-service.md (parent: auth)  │
    │                                        │
    │ Depth 2: (none found)                  │
    │                                        │
    └────────────────────────────────────────┘

Result Output (Tree Format):

    auth-service.md (depth: 0)
    ├── api-gateway.md (depth: 1)
    └── user-service.md (depth: 1)

Result Output (JSON Format):

    {
      "start_nodes": ["auth-service.md"],
      "direction": "backward",
      "nodes": [
        {
          "id": "auth-service.md",
          "depth": 0,
          "parents": [],
          "type": "document"
        },
        {
          "id": "api-gateway.md",
          "depth": 1,
          "parents": ["auth-service.md"],
          "type": "document"
        },
        {
          "id": "user-service.md",
          "depth": 1,
          "parents": ["auth-service.md"],
          "type": "document"
        }
      ],
      "edges": [
        {
          "source": "api-gateway.md",
          "target": "auth-service.md",
          "type": "dependency",
          "confidence": 1.0
        },
        {
          "source": "user-service.md",
          "target": "auth-service.md",
          "type": "dependency",
          "confidence": 1.0
        }
      ]
    }
```

### Component Detection Pipeline

The system automatically detects components (microservices, libraries) from documentation structure.

```
Graph with Heuristic Scores:

    api-gateway.md
    ├── In-degree: 0
    ├── Out-degree: 5
    ├── Filename heuristic: "api-gateway" contains "gateway" ✓
    └── Endpoint mentions: [GET /api, POST /api]
        Score: HIGH (named file + high out-degree)

    auth-service.md
    ├── In-degree: 2
    ├── Out-degree: 3
    ├── Filename heuristic: "auth-service" contains "service" ✓
    └── Endpoint mentions: [POST /auth/login, GET /auth/verify]
        Score: HIGH (named file)

    user-service.md
    ├── In-degree: 1
    ├── Out-degree: 2
    ├── Filename heuristic: "user-service" contains "service" ✓
    └── Endpoint mentions: [GET /users/{id}, POST /users]
        Score: HIGH (named file)

    redis
    ├── In-degree: 3
    ├── Out-degree: 0
    ├── Filename heuristic: None (no file)
    └── No endpoints mentioned
        Score: MEDIUM (high in-degree from detection)

Result:

    Detected Components:
    ├── api-gateway (type: service)
    │   └── Files: api-gateway.md
    │       Endpoints: GET /api, POST /api
    │       Dependencies: [auth-service, user-service]
    │
    ├── auth-service (type: service)
    │   └── Files: auth-service.md
    │       Endpoints: POST /auth/login, GET /auth/verify
    │       Dependencies: [redis, jwt-lib]
    │
    ├── user-service (type: service)
    │   └── Files: user-service.md
    │       Endpoints: GET /users/{id}, POST /users
    │       Dependencies: [auth-service, redis]
    │
    └── redis (type: external_dependency)
        └── Dependencies: None
            Used by: [auth-service, user-service]
```

### Agent Workflow: Complete Search → Crawl → Context Assembly

Multi-step workflow for agents to understand architecture:

```
┌──────────────────────────────────────────────────────────────┐
│ Agent: "How does authentication flow through the system?"    │
└──────────────────────────────────────────────────────────────┘
    ↓
Step 1: Full-Text Search

    Command: bmd query "authentication flow" --dir ./docs --format json

    BM25 Results:
    ├── [1] auth-service.md (score: 8.3)
    │       "OAuth2 authentication and JWT token validation"
    │
    ├── [2] api-gateway.md (score: 4.1)
    │       "authenticates requests via middleware"
    │
    └── [3] user-service.md (score: 2.7)
            "user authentication and permissions"

    ↓ Agent filters top results

Step 2: Graph Crawl from Relevant Files

    Command: bmd crawl --from-multiple auth-service.md,api-gateway.md \
                       --direction both --depth 3 --format json

    Discovered Graph:

    Backward (incoming to auth-service):
    ├── api-gateway.md ──calls──> auth-service.md
    │   "POST /auth/token validates with OAuth2 service"
    │
    └── user-service.md ──uses──> auth-service.md
        "validates user tokens via auth service"

    Forward (outgoing from auth-service):
    ├── auth-service.md ──depends──> redis
    │   "caches validated tokens"
    │
    ├── auth-service.md ──depends──> jwt-library
    │   "signs and verifies JWT tokens"
    │
    └── auth-service.md ──calls──> user-service.md
        "checks user permissions after auth"

    ↓
Step 3: Assemble RAG Context

    Command: bmd context "authentication flow" --dir ./docs --format json

    Assembled Context Block:

    ┌─────────────────────────────────────────────────────────┐
    │ Relevant Sections for "authentication flow":            │
    │                                                         │
    │ Source: auth-service.md                                 │
    │ ─────────────────────────────────                       │
    │ # OAuth2 Service                                        │
    │                                                         │
    │ Handles token validation and JWT signing for the       │
    │ entire system. All requests must pass through this     │
    │ service before accessing other microservices.          │
    │                                                         │
    │ § Architecture                                          │
    │ - Validates JWT tokens (kid, signature, expiry)        │
    │ - Caches valid tokens in redis (TTL: 24h)              │
    │ - Returns 401 for invalid tokens                       │
    │                                                         │
    │ Source: api-gateway.md                                  │
    │ ──────────────────────────                              │
    │ # API Gateway                                           │
    │                                                         │
    │ § Request Flow                                          │
    │ 1. Request arrives at /api/*                           │
    │ 2. AuthMiddleware calls auth-service                   │
    │ 3. If valid, forward to target service                 │
    │ 4. Return 401 if auth-service rejects                  │
    │                                                         │
    │ Source: user-service.md                                 │
    │ ─────────────────────────────                           │
    │ # User Service                                          │
    │                                                         │
    │ § Permission Checking                                   │
    │ After auth-service validates token, user-service       │
    │ checks role-based access for the requested resource.   │
    │                                                         │
    └─────────────────────────────────────────────────────────┘

    ↓
Step 4: Agent Reasoning

    With full context: search results + graph relationships + assembled docs
    → Agent understands complete authentication architecture
    → Can answer: flow, dependencies, failure points, optimization opportunities

    Example reasoning:
    "Authentication happens in 3 stages:
    1. API Gateway receives request
    2. Calls auth-service to validate token
    3. Auth-service verifies JWT and checks redis cache
    4. If valid, user-service applies role-based access control

    Critical path: request → api-gateway → auth-service → redis
    Optimization: implement token caching in api-gateway to reduce auth-service calls"
```

## Design Decisions

### Terminal-Only, No GUI
Keeps developers in the CLI workflow. ANSI rendering provides beautiful output in any terminal.

### Goldmark Parser
Extensible, GFM-compatible, well-maintained, allows custom renderers.

### Internal AST Abstraction
Isolates renderer from goldmark dependency, enables custom markup handling.

### Bubbletea for TUI
Standard Go choice for terminal UIs, event-driven architecture, good community.

### SQLite for Persistence
Fast, zero-config, WAL mode for concurrent reads, single-file database.

### BM25 Full-Text Search
Proven ranking algorithm, configurable parameters, efficient for medium-sized corpora.

### Atomic File Writes
Write to temp file, then rename ensures data durability and prevents corruption.

### Vim-Like Keybindings
Familiar to terminal-savvy developers, efficient navigation patterns.

## Performance

Benchmarks on 100-document corpus:

| Operation | Time |
|-----------|------|
| Index build | 44ms |
| Full-text search | <8ms |
| Keyword lookup | 3ms |
| Component detection | 18ms |
| Dependency query | 17ms |
| Split-pane rendering | <3ms |
| Graph crawl (50 nodes) | <1ms |

## Export & Import Infrastructure

**Goal:** Package knowledge artifacts (markdown + indexes + graphs) as portable tar files for container deployment

- `bmd export --from <dir>` — Package markdown + .bmd indexes into knowledge.tar.gz
- `bmd import <tar> --dir <dest>` — Extract with SHA256 checksum validation
- `bmd serve --headless --knowledge-tar <tar>` — Run MCP-only mode (no TUI) for containers
- Git provenance metadata (version, tag, commit, remote URL)
- S3 publish/download integration

**Files:** `internal/knowledge/export.go`, `internal/knowledge/importtar.go`

## Container Deployment

**Goal:** Deploy BMD as sidecar service for agent fleets

- **Dockerfile** — Multi-stage build, <30MB image, alpine-based
- **docker-compose.yml** — Sidecar pattern with health checks, resource limits
- **Kubernetes manifests** — Deployment (w/ InitContainer), Service, ConfigMap, Namespace

**Files:** `Dockerfile`, `docker-compose.yaml`, `kubernetes/` (5 manifests)

## Component Registry

**Goal:** Hybrid confidence-weighted dependency discovery combining link, mention, and LLM signals

### Signal Aggregation Architecture

```
Document Corpus
    ↓
┌─ Link Extractor (existing graph edges, confidence=1.0)
├─ Text Mention Extractor (pattern matching, confidence=0.60-0.75)
└─ LLM Extractor (PageIndex reasoning, confidence=0.65, opt-in)
    ↓
ComponentRegistry
    ├── Components map[ID → *RegistryComponent]
    └── Relationships []RegistryRelationship
              └── Signals []Signal (SourceType, Confidence, Evidence, Weight)
    ↓
HybridBuilder.BuildHybridGraph()
    ↓
Augmented Graph (higher confidence edges, new mention/LLM edges)
    ↓
CLI Commands
    ├── bmd depends --from/--to (relationship queries)
    ├── bmd relationships (signal-aware queries)
    ├── bmd components list/search/inspect
    └── bmd depends (enriched with hybrid signals)
```

### Aggregation Strategy

Default: **AggregationMax** — `max(signal.confidence * signal.weight)` across all signals.

Rationale: Conservative, predictable, well-behaved with extreme signal weights. A link (1.0) always wins.

Alternative: **AggregationWeightedAverage** available for callers that need weighted consensus.

### Mention Pattern Library

Text mentions use a confidence-weighted pattern library:
- `0.75`: "depends on X", "calls X service", "uses the X"
- `0.70`: "connects to X", "communicates with X", "integrates with X"
- `0.65`: generic prose mentions of known component names
- `0.60`: weak signal (component name appears but context unclear)

### Data Flow

```
bmd depends --from auth-service --format json
    ↓
ParseDependsArgs → loadOrBuildRegistry
    ↓ (cache hit)
LoadRegistry(".bmd-registry.json") → ComponentRegistry
    ↓ (no cache — bootstrap)
loadGraphAndServices → InitFromGraph(graph, docs)
    [Link signals → Mention signals → LLM signals (optional)]
    ↓
FindRelationships("auth-service") → []RegistryRelationship
    ↓
marshalContract(NewOKResponse(...)) → CONTRACT-01 JSON
```

**Key files:**
- `internal/knowledge/registry.go` — ComponentRegistry, Signal, RegistryRelationship
- `internal/knowledge/mention_extractor.go` — Text mention extraction
- `internal/knowledge/mention_patterns.go` — Pattern confidence library
- `internal/knowledge/llm_extractor.go` — PageIndex LLM subprocess wrapper
- `internal/knowledge/hybrid_builder.go` — HybridBuilder, AggregateSignals, BuildHybridGraph
- `internal/knowledge/registry_cmd.go` — CLI commands (components list/search/inspect, relationships)

## Knowledge Versioning & Distribution

**Goal:** Enable knowledge artifacts as versioned, distributable assets

- SHA256 deterministic checksums (sorted archive paths)
- Semantic versioning (`--version 2.0`, `--git-version` auto-detect)
- Git provenance auto-detection (remote, tag, commit hash)
- S3 cloud distribution (`--publish s3://`, `import s3://`)
- Automatic checksum validation on import

**Files:** Extended `internal/knowledge/export.go` with version/checksum/S3 functions

### Live Graph Updates

Goal: Automatically keep knowledge index fresh as markdown files change

- **FileWatcher:** Polling-based (500ms interval, stdlib only, no fsnotify). Tracks .md file changes via mtime snapshot diff. sync.Once Stop() for idempotency.
- **IncrementalUpdater:** Delta-applies changes to BM25 index + knowledge graph + component registry. Handles WatchCreated, WatchModified, WatchDeleted events. Edge cleanup before re-extraction.
- **WatchSessionManager:** Isolates MCP watch sessions — multiple agents can watch concurrently.
- **CLI:** `bmd watch` — outputs events to stderr (JSON stdout unaffected)
- **MCP Tool:** bmd/watch session management via WatchSessionManager
- **Signal flow:** FileWatcher.Events → IncrementalUpdater.onChange → MCP notification

**Files:** `internal/knowledge/watcher.go`, `incremental_updater.go`, `watch_session.go`

### Intelligent Relationship Discovery

Goal: Go beyond explicit links to discover implicit component relationships from prose and config files

- **Semantic NER Extractor:** Named Entity Recognition over prose text to extract component names referenced in context (services, libraries, databases) — beyond text pattern matching
- **Manifest Parser:** Parses infrastructure files to extract structural dependencies:
  - docker-compose.yml → service depends_on relationships
  - Kubernetes manifests → service → configmap/secret/volume relationships
  - package.json → npm dependency graph
  - go.mod → Go module dependencies
- **Multi-Signal Fusion:** Merges semantic + manifest + link + mention signals with calibrated weights
  - Manifest signal: confidence 0.85 (structural, authoritative)
  - Semantic/NER signal: confidence 0.65-0.75 (inferred from prose)
- **Validation:** 63 relationships correctly discovered from 12-document test corpus, 100% accuracy
- **Architecture:** Additive signal layer; InitFromGraph + InitFromManifests + InitFromNER

**Files:** `internal/knowledge/semantic_extractor.go`, `manifest_parser.go`, `ner_extractor.go`

## Project Status

All 20 phases complete and production-ready:
- ✅ Rendering engine with syntax highlighting
- ✅ Full editor with persistence and undo/redo
- ✅ Navigation and link following
- ✅ Full-text search and BM25 indexing
- ✅ Knowledge graph and component detection
- ✅ Directory browser with split-pane view
- ✅ Image rendering support
- ✅ MCP server (`bmd serve --mcp`) for native agent integration
- ✅ Graph traversal with multi-start BFS and cycle detection
- ✅ **Export/import infrastructure with tar packaging**
- ✅ **Container deployment (Docker, Compose, Kubernetes)**
- ✅ **Knowledge versioning and S3 distribution**
- ✅ **Component registry with hybrid signal aggregation**
- ✅ **Live graph updates with file watching**
- ✅ **Intelligent relationship discovery from prose and manifests**
- ✅ **LLM-powered relationship validation with auto-accept/reject thresholds**
- ✅ 415+ unit tests

**Current Status:** Feature-complete. All phases shipped. Production-ready.
