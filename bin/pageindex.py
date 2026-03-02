#!/usr/bin/env python3
import json
import sys
import argparse
import re
from pathlib import Path
from typing import Any, Dict, List, Tuple

def main():
    parser = argparse.ArgumentParser(prog='pageindex')
    subparsers = parser.add_subparsers(dest='command', required=True)

    # index subcommand
    index_parser = subparsers.add_parser('index')
    index_parser.add_argument('--file', required=True, help='File path to index')
    index_parser.add_argument('--model', default='claude-sonnet-4-5', help='LLM model to use')
    index_parser.add_argument('--format', default='json', help='Output format')

    # query subcommand
    query_parser = subparsers.add_parser('query')
    query_parser.add_argument('--query', required=True, help='Search query')
    query_parser.add_argument('--model', default='claude-sonnet-4-5', help='LLM model to use')
    query_parser.add_argument('--top', type=int, default=10, help='Number of results')
    query_parser.add_argument('--format', default='json', help='Output format')

    try:
        args = parser.parse_args()
    except SystemExit as e:
        print(json.dumps({"error": f"Argument parsing failed: {e}"}), file=sys.stderr)
        sys.exit(1)

    if args.command == 'index':
        handle_index(args)
    elif args.command == 'query':
        handle_query(args)

def parse_markdown_sections(content: str, file_path: str) -> Dict[str, Any]:
    """
    Parse markdown file into sections by heading level.
    Returns a tree structure with root containing all content.
    """
    lines = content.split('\n')
    heading_pattern = re.compile(r'^(#{1,6})\s+(.+)$')

    # Collect sections with their line ranges
    sections = []
    heading_indices = []

    # Find all heading lines
    for i, line in enumerate(lines):
        match = heading_pattern.match(line)
        if match:
            level = len(match.group(1))
            text = match.group(2).strip()
            heading_indices.append((i, level, text))

    # Extract sections between headings
    if not heading_indices:
        # No headings, entire file is one section
        sections.append({
            'heading': '',
            'level': 0,
            'line_start': 0,
            'line_end': len(lines) - 1,
            'content': content,
            'children': []
        })
    else:
        # Add content before first heading if any
        if heading_indices[0][0] > 0:
            before_first = '\n'.join(lines[:heading_indices[0][0]])
            sections.append({
                'heading': '',
                'level': 0,
                'line_start': 0,
                'line_end': heading_indices[0][0] - 1,
                'content': before_first.strip(),
                'children': []
            })

        # Add sections for each heading
        for idx, (line_num, level, heading_text) in enumerate(heading_indices):
            # Find end of this section (start of next heading at same or higher level)
            end_line = len(lines) - 1
            for next_idx in range(idx + 1, len(heading_indices)):
                next_line, next_level, _ = heading_indices[next_idx]
                if next_level <= level:
                    end_line = next_line - 1
                    break

            # Extract section content
            section_lines = lines[line_num:end_line + 1]
            section_content = '\n'.join(section_lines).strip()

            sections.append({
                'heading': heading_text,
                'level': level,
                'line_start': line_num,
                'line_end': end_line,
                'content': section_content,
                'children': []
            })

    # Build hierarchical structure
    root = {
        'heading': '',
        'level': 0,
        'line_start': 0,
        'line_end': len(lines) - 1,
        'content': content,
        'children': []
    }

    # Build tree from flat sections
    if sections:
        root['children'] = build_tree(sections)

    return {
        'file': file_path,
        'root': root
    }

def build_tree(sections: List[Dict]) -> List[Dict]:
    """Build hierarchical tree from flat section list."""
    if not sections:
        return []

    result = []
    stack = []  # Stack of (section, level)

    for section in sections:
        level = section['level']

        # Pop sections with equal or higher level from stack
        while stack and stack[-1][1] >= level:
            stack.pop()

        # Create tree node (without 'level' and 'children' will be added as we go)
        node = {
            'heading': section['heading'],
            'summary': section['content'][:100] if section['content'] else '',  # First 100 chars for summary
            'content': section['content'],
            'line_start': section['line_start'],
            'line_end': section['line_end'],
            'children': []
        }

        # Add to parent or root
        if stack:
            stack[-1][0]['children'].append(node)
        else:
            result.append(node)

        stack.append((node, level))

    return result

def handle_index(args):
    file_path = args.file

    # Validate file exists
    if not Path(file_path).exists():
        print(json.dumps({"error": f"File not found: {file_path}"}), file=sys.stderr)
        sys.exit(1)

    try:
        # Read the markdown file
        with open(file_path, 'r', encoding='utf-8') as f:
            content = f.read()

        # Parse into sections
        tree = parse_markdown_sections(content, file_path)

        if args.format == 'json':
            print(json.dumps(tree))
        else:
            print(f"Format {args.format} not supported", file=sys.stderr)
            sys.exit(1)

    except Exception as e:
        print(json.dumps({"error": f"{type(e).__name__}: {str(e)}"}), file=sys.stderr)
        sys.exit(1)

def walk_sections(node: Dict[str, Any], file_path: str = "") -> List[Dict[str, Any]]:
    """
    Recursively walk section tree and return flat list of all sections
    with their heading paths and content.
    """
    results = []

    def traverse(section, heading_path_parts):
        # Build heading_path
        if section.get('heading'):
            new_path = heading_path_parts + [section['heading']]
        else:
            new_path = heading_path_parts

        section_copy = {
            'file': file_path,
            'heading': section.get('heading', ''),
            'heading_path': ' > '.join(new_path) if new_path else '',
            'content': section.get('content', ''),
            'line_start': section.get('line_start', 0),
            'line_end': section.get('line_end', 0),
        }

        results.append(section_copy)

        # Recurse into children
        for child in section.get('children', []):
            traverse(child, new_path)

    traverse(node, [])
    return results

def score_section(section: Dict[str, Any], query: str) -> float:
    """
    Score a section based on keyword matches in its content.
    Case-insensitive scoring: +1 for each keyword occurrence.
    """
    content_lower = section.get('content', '').lower()
    heading_lower = section.get('heading', '').lower()
    query_lower = query.lower()

    # Split query into keywords
    keywords = query_lower.split()

    score = 0.0

    # Check each keyword
    for keyword in keywords:
        # Count occurrences in content
        content_count = content_lower.count(keyword)
        # Heading matches worth more
        heading_count = (heading_lower.count(keyword) * 2)

        score += content_count + heading_count

    return float(score)

def handle_query(args):
    """Handle query subcommand - reads trees from stdin and performs semantic search"""
    try:
        # Read trees from stdin
        stdin_data = sys.stdin.read()

        if not stdin_data.strip():
            trees = []
        else:
            trees = json.loads(stdin_data)

        query = args.query
        top = args.top

        # Collect all sections from all trees with their scores
        all_results = []

        if isinstance(trees, list):
            for tree in trees:
                file_path = tree.get("file", "")
                if "root" in tree:
                    sections = walk_sections(tree["root"], file_path)

                    for section in sections:
                        score = score_section(section, query)

                        # Only include sections with non-zero score
                        if score > 0:
                            result = {
                                "file": section['file'],
                                "heading_path": section.get("heading_path", ""),
                                "content": section.get("content", ""),
                                "score": score,
                                "reasoning_trace": f"Matched query '{query}' in section" +
                                    (f" '{section.get('heading', '')}'" if section.get('heading') else "")
                            }
                            all_results.append(result)

        # Sort by score descending
        all_results.sort(key=lambda x: x["score"], reverse=True)

        # Limit to top N
        results = all_results[:top]

        if args.format == 'json':
            # Return array directly (not wrapped in object)
            print(json.dumps(results))
        else:
            print(f"Format {args.format} not supported", file=sys.stderr)
            sys.exit(1)

    except json.JSONDecodeError as e:
        print(json.dumps({"error": f"Invalid JSON in stdin: {e}"}), file=sys.stderr)
        sys.exit(1)
    except Exception as e:
        print(json.dumps({"error": f"{type(e).__name__}: {str(e)}"}), file=sys.stderr)
        sys.exit(1)

if __name__ == "__main__":
    main()
