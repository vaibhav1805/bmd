# Wave 2: Complete Block-Level Rendering Test

This is the comprehensive test document for Phase 1 Wave 2. It exercises every block-level
markdown element alongside the inline formatting from Wave 1.

## Inline Formatting Recap

Before diving into block elements, let's verify Wave 1 features still work:
**bold text**, *italic text*, ***bold and italic***, ~~strikethrough~~, and `inline code`.

Here is a longer sentence with inline elements woven throughout: The **most important** thing
is that *each style* renders distinctly from the others, with `code()` always having a
background color.

## Headings at Every Level

### H3: A Subsection Heading

This paragraph sits under an H3 heading. The heading should be visually smaller than H1
and H2, but still distinct from body text.

#### H4: A Deeper Subsection

H4 headings use italic style to differentiate them further.

##### H5: Even Deeper

H5 with italic and a lighter color.

###### H6: Maximum Depth

H6 is the least prominent heading level.

## Lists

### Unordered List with Nesting

- First top-level item
- Second top-level item with **bold** and `code` inline
  - Nested item A
  - Nested item B with *italic*
    - Doubly nested item
- Third top-level item

### Ordered List with Nesting

1. First ordered item
2. Second ordered item
   1. Nested first step
   2. Nested second step
3. Third ordered item
4. Fourth ordered item

### Mixed List Example

- Bullet item one: use `go build` to compile
- Bullet item two: run `./bmd file.md` to render
- Bullet item three: check the output visually

## Blockquotes

> "The best way to predict the future is to invent it."
> — Alan Kay

Here is a multi-paragraph blockquote to test border continuity:

> This is the opening paragraph of a longer blockquote. It spans multiple
> sentences and should display with the left border character on every line.
>
> This is the second paragraph within the same blockquote. The border should
> continue here without any gaps or misalignment.

A blockquote can also contain inline formatting:

> **Important:** Always read the *documentation* before filing an issue.
> Refer to the `README.md` for getting started.

## Code Blocks

### Python Example

```python
def fibonacci(n):
    """Return the nth Fibonacci number."""
    if n <= 1:
        return n
    a, b = 0, 1
    for _ in range(2, n + 1):
        a, b = b, a + b
    return b

# Print first 10 Fibonacci numbers
for i in range(10):
    print(f"fib({i}) = {fibonacci(i)}")
```

### Go Example

```go
package main

import (
    "fmt"
    "os"
)

func main() {
    if len(os.Args) < 2 {
        fmt.Fprintln(os.Stderr, "usage: bmd <file.md>")
        os.Exit(1)
    }
    fmt.Printf("Rendering: %s\n", os.Args[1])
}
```

### JavaScript Example

```javascript
async function fetchData(url) {
    try {
        const response = await fetch(url);
        if (!response.ok) {
            throw new Error(`HTTP error: ${response.status}`);
        }
        return await response.json();
    } catch (err) {
        console.error('Fetch failed:', err.message);
        return null;
    }
}
```

### Code Block Without Language Label

```
This is a plain code block with no language annotation.
It should render without syntax highlighting but still
display inside the box-drawing border.

    Indented lines preserve their indentation.
```

## Tables

### Simple Data Table

| Name    | Language   | Year |
|---------|------------|------|
| Go      | Compiled   | 2009 |
| Python  | Interpreted| 1991 |
| Rust    | Compiled   | 2010 |
| JavaScript | Interpreted | 1995 |

### Alignment Test Table

| Left Aligned | Center Aligned | Right Aligned |
|:-------------|:--------------:|--------------:|
| apple        | banana         | cherry        |
| dog          | elephant       | fox           |
| gold         | silver         | bronze        |

### Feature Comparison Table

| Feature             | bmd  | cat  | less |
|---------------------|------|------|------|
| Syntax Highlighting | Yes  | No   | No   |
| Bold/Italic         | Yes  | No   | No   |
| Tables              | Yes  | No   | No   |
| Terminal Native     | Yes  | Yes  | Yes  |
| Scrolling           | No   | No   | Yes  |

## Mixed Content Section

This section mixes multiple block types to ensure they all work together.

> **Pro Tip:** Combine lists and code for documentation that reads like a tutorial.

Steps to build the project:

1. Clone the repository
2. Install dependencies: `go mod download`
3. Build the binary:

```bash
go build -o bmd ./cmd/bmd
```

4. Run on a markdown file:

```bash
./bmd README.md
```

5. The output should be styled — headings in color, code highlighted, tables aligned.

---

## End of Test Document

If all of the above renders correctly — headings with visual hierarchy, colored and
bulleted lists, blockquotes with left borders, code blocks with syntax highlighting and
box-drawing borders, and tables with aligned columns — then Phase 1 Wave 2 is complete.
