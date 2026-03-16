/*
 * Copyright 2025 ynet-dev Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import {
  SourceFile,
  ChineseComment,
  ParsedComment,
  FileWithComments,
  CommentType,
  MultiLineContext,
} from '../types/index';
import { getCommentPatterns } from '../utils/language';
import { containsChinese, cleanCommentText } from '../utils/chinese';

/**
 * Checks if the specified location is inside a string literal
 */
const isInsideStringLiteral = (line: string, position: number): boolean => {
  let insideDoubleQuote = false;
  let insideSingleQuote = false;
  let insideBacktick = false;
  let escapeNext = false;

  for (let i = 0; i < position; i++) {
    const char = line[i];

    if (escapeNext) {
      escapeNext = false;
      continue;
    }

    if (char === '\\') {
      escapeNext = true;
      continue;
    }

    if (char === '"' && !insideSingleQuote && !insideBacktick) {
      insideDoubleQuote = !insideDoubleQuote;
    } else if (char === "'" && !insideDoubleQuote && !insideBacktick) {
      insideSingleQuote = !insideSingleQuote;
    } else if (char === '`' && !insideDoubleQuote && !insideSingleQuote) {
      insideBacktick = !insideBacktick;
    }
  }

  return insideDoubleQuote || insideSingleQuote || insideBacktick;
};

/**
 * Parsing single-line comments
 */
const parseSingleLineComments = (
  content: string,
  pattern: RegExp,
  language?: string,
): ParsedComment[] => {
  const comments: ParsedComment[] = [];
  const lines = content.split('\n');

  // Add a security check
  const maxLines = 5000; // Down to 5000 lines
  if (lines.length > maxLines) {
    console.warn(`⚠️  文件行数过多 (${lines.length}行)，跳过单行注释解析`);
    return comments;
  }

  lines.forEach((line, index) => {
    pattern.lastIndex = 0; // Reset regular expression index
    let match: RegExpExecArray | null;

    // Find all matches, but keep only those not in the string
    let matchCount = 0;
    const maxMatches = 100; // Limit each line to a maximum of 100 matches
    let lastIndex = 0;

    while ((match = pattern.exec(line)) !== null) {
      // Multiple protections against infinite loops
      matchCount++;
      if (matchCount > maxMatches) {
        console.warn(
          `⚠️  单行匹配次数过多，中断处理: ${line.substring(0, 50)}...`,
        );
        break;
      }

      // Check if lastIndex is advancing to prevent an infinite loop
      if (pattern.global) {
        if (pattern.lastIndex <= lastIndex) {
          // If lastIndex does not advance, manually advance one bit to avoid infinite loops
          pattern.lastIndex = lastIndex + 1;
          if (pattern.lastIndex >= line.length) {
            break;
          }
        }
        lastIndex = pattern.lastIndex;
      }

      if (match[1]) {
        const commentContent = match[1];
        let commentStartIndex = match.index!;
        let commentLength = 2; // Default is//

        // Determine annotation symbols based on language
        if (
          language === 'yaml' ||
          language === 'toml' ||
          language === 'shell' ||
          language === 'python' ||
          language === 'ruby'
        ) {
          commentStartIndex = line.indexOf('#', match.index!);
          commentLength = 1; // #length is 1
        } else if (language === 'ini') {
          // INI files may use #or;
          const hashIndex = line.indexOf('#', match.index!);
          const semicolonIndex = line.indexOf(';', match.index!);
          if (
            hashIndex >= 0 &&
            (semicolonIndex < 0 || hashIndex < semicolonIndex)
          ) {
            commentStartIndex = hashIndex;
            commentLength = 1;
          } else if (semicolonIndex >= 0) {
            commentStartIndex = semicolonIndex;
            commentLength = 1;
          }
        } else if (language === 'php') {
          // PHP may use//or #
          const slashIndex = line.indexOf('//', match.index!);
          const hashIndex = line.indexOf('#', match.index!);
          if (slashIndex >= 0 && (hashIndex < 0 || slashIndex < hashIndex)) {
            commentStartIndex = slashIndex;
            commentLength = 2;
          } else if (hashIndex >= 0) {
            commentStartIndex = hashIndex;
            commentLength = 1;
          }
        } else {
          // JavaScript/TypeScript/Go/Java/C/C++/C# style
          commentStartIndex = line.indexOf('//', match.index!);
          commentLength = 2;
        }

        const startColumn = commentStartIndex;
        const endColumn = startColumn + commentLength + commentContent.length;

        // Check if the comment starts inside the string
        if (
          commentStartIndex >= 0 &&
          !isInsideStringLiteral(line, commentStartIndex)
        ) {
          comments.push({
            content: commentContent,
            startLine: index + 1,
            endLine: index + 1,
            startColumn,
            endColumn,
            type: 'single-line',
          });
        }
      }

      // Prevent infinite loops
      if (!pattern.global) break;
    }
  });

  return comments;
};

/**
 * Parse multiline comments
 */
const parseMultiLineComments = (
  content: string,
  startPattern: RegExp,
  endPattern: RegExp,
): ParsedComment[] => {
  const comments: ParsedComment[] = [];
  const lines = content.split('\n');
  let inComment = false;
  let commentStart: { line: number; column: number } | null = null;
  let commentLines: string[] = [];

  // Add a security check
  const maxLines = 5000; // Down to 5000 lines
  if (lines.length > maxLines) {
    console.warn(`⚠️  文件行数过多 (${lines.length}行)，跳过多行注释解析`);
    return comments;
  }

  // Add processing counters to prevent infinite loops
  let processedLines = 0;
  const maxProcessedLines = 10000;

  for (let lineIndex = 0; lineIndex < lines.length; lineIndex++) {
    const line = lines[lineIndex];

    // Prevent unlimited processing
    processedLines++;
    if (processedLines > maxProcessedLines) {
      console.warn(`⚠️  处理行数超限，中断解析`);
      break;
    }

    if (!inComment) {
      startPattern.lastIndex = 0;
      const startMatch = startPattern.exec(line);

      if (startMatch && !isInsideStringLiteral(line, startMatch.index!)) {
        inComment = true;
        commentStart = { line: lineIndex + 1, column: startMatch.index! };

        // Check if they end on the same line
        endPattern.lastIndex = startMatch.index! + startMatch[0].length;
        const endMatch = endPattern.exec(line);

        if (endMatch) {
          // single-line multi-line comment
          const commentContent = line.substring(
            startMatch.index! + startMatch[0].length,
            endMatch.index!,
          );

          comments.push({
            content: commentContent,
            startLine: lineIndex + 1,
            endLine: lineIndex + 1,
            startColumn: startMatch.index!,
            endColumn: endMatch.index! + endMatch[0].length,
            type: 'multi-line',
          });

          inComment = false;
          commentStart = null;
        } else {
          // Start with a multi-line comment
          const commentContent = line.substring(
            startMatch.index! + startMatch[0].length,
          );
          commentLines = [commentContent];
        }
      }
    } else {
      // In a multi-line comment
      endPattern.lastIndex = 0;
      const endMatch = endPattern.exec(line);

      if (endMatch) {
        // End of multiline comment
        const commentContent = line.substring(0, endMatch.index!);
        commentLines.push(commentContent);

        comments.push({
          content: commentLines.join('\n'),
          startLine: commentStart!.line,
          endLine: lineIndex + 1,
          startColumn: commentStart!.column,
          endColumn: endMatch.index! + endMatch[0].length,
          type: 'multi-line',
        });

        inComment = false;
        commentStart = null;
        commentLines = [];
      } else {
        // Continue with multi-line comments
        commentLines.push(line);
      }
    }
  }

  return comments;
};

/**
 * Parse all comments in the file
 */
export const parseComments = (file: SourceFile): ParsedComment[] => {
  const patterns = getCommentPatterns(file.language);
  if (!patterns) return [];

  const singleLineComments = parseSingleLineComments(
    file.content,
    patterns.single,
    file.language,
  );
  const multiLineComments = parseMultiLineComments(
    file.content,
    patterns.multiStart,
    patterns.multiEnd,
  );

  return [...singleLineComments, ...multiLineComments];
};

/**
 * Filter comments containing Chinese and process multi-line comments line by line
 */
export const filterChineseComments = (
  comments: ParsedComment[],
  language?: string,
): ChineseComment[] => {
  const result: ChineseComment[] = [];

  for (const comment of comments) {
    if (comment.type === 'multi-line' && comment.content.includes('\n')) {
      // Multi-line comments: line-by-line processing
      const multiLineResults = processMultiLineCommentForChinese(
        comment,
        language,
      );
      result.push(...multiLineResults);
    } else if (containsChinese(comment.content)) {
      // Single-line comments or single-line multi-line comments
      result.push({
        ...comment,
        content: cleanCommentText(
          comment.content,
          comment.type === 'documentation' ? 'multi-line' : comment.type,
          language,
        ),
      });
    }
  }

  return result;
};

/**
 * Processing multi-line comments, extracting lines containing Chinese as independent comment units
 */
const processMultiLineCommentForChinese = (
  comment: ParsedComment,
  language?: string,
): ChineseComment[] => {
  const lines = comment.content.split('\n');
  const result: ChineseComment[] = [];

  lines.forEach((line, lineIndex) => {
    const cleanedLine = cleanCommentText(line, 'multi-line', language);

    if (containsChinese(cleanedLine)) {
      // Calculate the position of this line in the original file
      const actualLineNumber = comment.startLine + lineIndex;

      // Create a comment object representing this line
      const lineComment: ChineseComment = {
        content: cleanedLine,
        startLine: actualLineNumber,
        endLine: actualLineNumber,
        startColumn: 0, // This value needs to be calculated more precisely, but for line processing within a multi-line comment, use 0 for the time being.
        endColumn: line.length,
        type: 'multi-line',
        // Add metadata with multi-line comments for subsequent processing
        multiLineContext: {
          isPartOfMultiLine: true,
          originalComment: comment,
          lineIndexInComment: lineIndex,
          totalLinesInComment: lines.length,
        },
      };

      result.push(lineComment);
    }
  });

  return result;
};

/**
 * Detect Chinese comments in files
 */
export const detectChineseInFile = (file: SourceFile): ChineseComment[] => {
  try {
    // Simple protection: skipping large files
    if (file.content.length > 500000) {
      // 500KB
      console.warn(
        `⚠️  跳过大文件: ${file.path} (${file.content.length} 字符)`,
      );
      return [];
    }

    // Simple protection: skip files with too many lines
    const lines = file.content.split('\n');
    if (lines.length > 10000) {
      console.warn(`⚠️  跳过多行文件: ${file.path} (${lines.length} 行)`);
      return [];
    }

    const allComments = parseComments(file);
    return filterChineseComments(allComments, file.language);
  } catch (error) {
    console.error(`❌ 文件处理失败: ${file.path} - ${error}`);
    return [];
  }
};

/**
 * Batch detection of Chinese comments in multiple files
 */
export const detectChineseInFiles = (
  files: SourceFile[],
): FileWithComments[] => {
  const results: FileWithComments[] = [];

  for (let i = 0; i < files.length; i++) {
    const file = files[i];
    const fileName = file.path.split('/').pop() || file.path;

    console.log(`🔍 检测进度: ${i + 1}/${files.length} (当前: ${fileName})`);

    try {
      const chineseComments = detectChineseInFile(file);

      if (chineseComments.length > 0) {
        results.push({
          file,
          chineseComments,
        });
      }

      console.log(
        `✅ 完成: ${fileName} (找到 ${chineseComments.length} 条中文注释)`,
      );
    } catch (error) {
      console.error(`❌ 处理文件失败: ${fileName} - ${error}`);
      // Continue working on other documents
      continue;
    }
  }

  return results;
};

/**
 * Get annotation statistics
 */
export const getCommentStats = (
  files: SourceFile[],
): {
  totalFiles: number;
  filesWithComments: number;
  totalComments: number;
  chineseComments: number;
  commentsByType: Record<CommentType, number>;
} => {
  let totalComments = 0;
  let chineseComments = 0;
  let filesWithComments = 0;
  const commentsByType: Record<CommentType, number> = {
    'single-line': 0,
    'multi-line': 0,
    documentation: 0,
  };

  files.forEach(file => {
    const allComments = parseComments(file);
    const chineseCommentsInFile = filterChineseComments(
      allComments,
      file.language,
    );

    if (chineseCommentsInFile.length > 0) {
      filesWithComments++;
    }

    totalComments += allComments.length;
    chineseComments += chineseCommentsInFile.length;

    chineseCommentsInFile.forEach(comment => {
      commentsByType[comment.type]++;
    });
  });

  return {
    totalFiles: files.length,
    filesWithComments,
    totalComments,
    chineseComments,
    commentsByType,
  };
};
