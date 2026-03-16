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
  ProcessingReport,
  ProcessingStats,
  FileProcessingDetail,
} from '../types/index.js';

/**
 * report collector class
 */
export class ReportCollector {
  private stats: ProcessingStats = {
    totalFiles: 0,
    processedFiles: 0,
    translatedComments: 0,
    skippedFiles: 0,
    errors: [],
    startTime: Date.now(),
    endTime: 0,
  };

  private fileDetails: Map<string, FileProcessingDetail> = new Map();

  /**
   * Record file processing begins
   */
  recordFileStart(filePath: string): void {
    this.stats.totalFiles++;
    this.fileDetails.set(filePath, {
      file: filePath,
      commentCount: 0,
      status: 'processing',
      startTime: Date.now(),
    });
  }

  /**
   * Record file processing completed
   */
  recordFileComplete(filePath: string, commentCount: number): void {
    const detail = this.fileDetails.get(filePath);
    if (detail) {
      detail.status = 'success';
      detail.commentCount = commentCount;
      detail.endTime = Date.now();
      this.stats.processedFiles++;
      this.stats.translatedComments += commentCount;
    }
  }

  /**
   * log file skip
   */
  recordFileSkipped(filePath: string, reason?: string): void {
    const detail = this.fileDetails.get(filePath);
    if (detail) {
      detail.status = 'skipped';
      detail.errorMessage = reason;
      detail.endTime = Date.now();
      this.stats.skippedFiles++;
    }
  }

  /**
   * Log processing errors
   */
  recordError(filePath: string, error: Error): void {
    const detail = this.fileDetails.get(filePath);
    if (detail) {
      detail.status = 'error';
      detail.errorMessage = error.message;
      detail.endTime = Date.now();
    }
    this.stats.errors.push({ file: filePath, error: error.message });
  }

  /**
   * Complete statistics
   */
  finalize(): void {
    this.stats.endTime = Date.now();
  }

  /**
   * Obtain statistical information
   */
  getStats(): ProcessingStats {
    return { ...this.stats };
  }

  /**
   * Get file details
   */
  getFileDetails(): FileProcessingDetail[] {
    return Array.from(this.fileDetails.values());
  }

  /**
   * Generate a full report
   */
  generateReport(): ProcessingReport {
    this.finalize();
    const duration = (this.stats.endTime - this.stats.startTime) / 1000;

    return {
      stats: this.getStats(),
      details: this.getFileDetails(),
      duration,
    };
  }

  /**
   * Reset collector
   */
  reset(): void {
    this.stats = {
      totalFiles: 0,
      processedFiles: 0,
      translatedComments: 0,
      skippedFiles: 0,
      errors: [],
      startTime: Date.now(),
      endTime: 0,
    };
    this.fileDetails.clear();
  }
}

/**
 * Generate console reports
 */
export const generateConsoleReport = (report: ProcessingReport): string => {
  const { stats, duration } = report;
  const successRate =
    stats.totalFiles > 0
      ? ((stats.processedFiles / stats.totalFiles) * 100).toFixed(1)
      : '0';

  let output = `
📊 翻译处理报告
==================
总文件数: ${stats.totalFiles}
处理成功: ${stats.processedFiles}
跳过文件: ${stats.skippedFiles}
翻译注释: ${stats.translatedComments}
错误数量: ${stats.errors.length}
成功率: ${successRate}%
处理时间: ${duration.toFixed(2)}秒
`;

  if (stats.errors.length > 0) {
    output += '\n❌ 错误详情:\n';
    stats.errors.forEach(error => {
      output += `  ${error.file}: ${error.error}\n`;
    });
  } else {
    output += '\n✅ 处理完成，无错误';
  }

  return output;
};

/**
 * Generating Markdown Reports
 */
export const generateMarkdownReport = (report: ProcessingReport): string => {
  const { stats, details, duration } = report;
  const successRate =
    stats.totalFiles > 0
      ? ((stats.processedFiles / stats.totalFiles) * 100).toFixed(1)
      : '0';

  let markdown = `# 中文注释翻译报告

## 📊 统计概览

| 指标 | 数值 |
|------|------|
| 总文件数 | ${stats.totalFiles} |
| 处理成功 | ${stats.processedFiles} |
| 跳过文件 | ${stats.skippedFiles} |
| 翻译注释 | ${stats.translatedComments} |
| 错误数量 | ${stats.errors.length} |
| 成功率 | ${successRate}% |
| 处理时间 | ${duration.toFixed(2)}秒 |

## 📁 文件详情

| 文件路径 | 状态 | 注释数量 | 耗时(ms) | 备注 |
|----------|------|----------|----------|------|
`;

  details.forEach(detail => {
    const duration =
      detail.endTime && detail.startTime
        ? detail.endTime - detail.startTime
        : 0;
    const status =
      detail.status === 'success'
        ? '✅'
        : detail.status === 'error'
        ? '❌'
        : detail.status === 'skipped'
        ? '⏭️'
        : '🔄';

    markdown += `| ${detail.file} | ${status} | ${
      detail.commentCount
    } | ${duration} | ${detail.errorMessage || '-'} |\n`;
  });

  if (stats.errors.length > 0) {
    markdown += '\n## ❌ 错误详情\n\n';
    stats.errors.forEach((error, index) => {
      markdown += `${index + 1}. **${error.file}**\n   \`\`\`\n   ${
        error.error
      }\n   \`\`\`\n\n`;
    });
  }

  return markdown;
};

/**
 * Generate JSON reports
 */
export const generateJsonReport = (report: ProcessingReport): string => {
  return JSON.stringify(report, null, 2);
};

/**
 * Generate reports according to the format
 */
export const generateReport = (
  report: ProcessingReport,
  format: 'json' | 'markdown' | 'console' = 'console',
): string => {
  switch (format) {
    case 'json':
      return generateJsonReport(report);
    case 'markdown':
      return generateMarkdownReport(report);
    case 'console':
    default:
      return generateConsoleReport(report);
  }
};

/**
 * Save report to file
 */
export const saveReportToFile = async (
  report: ProcessingReport,
  filePath: string,
  format: 'json' | 'markdown' | 'console' = 'json',
): Promise<void> => {
  const content = generateReport(report, format);
  const fs = await import('fs/promises');
  await fs.writeFile(filePath, content, 'utf-8');
};

/**
 * Display real-time progress on the console
 */
export class ProgressDisplay {
  private total: number = 0;
  private current: number = 0;
  private startTime: number = Date.now();

  constructor(total: number) {
    this.total = total;
  }

  /**
   * update progress
   */
  update(current: number, currentFile?: string): void {
    this.current = current;
    const percentage = ((current / this.total) * 100).toFixed(1);
    const elapsed = (Date.now() - this.startTime) / 1000;
    const speed = current / elapsed;
    const eta = speed > 0 ? (this.total - current) / speed : 0;

    let line = `进度: ${current}/${
      this.total
    } (${percentage}%) | 耗时: ${elapsed.toFixed(1)}s`;

    if (eta > 0) {
      line += ` | 预计剩余: ${eta.toFixed(1)}s`;
    }

    if (currentFile) {
      line += ` | 当前: ${currentFile}`;
    }

    // Clear the current line and output the new progress
    process.stdout.write(
      '\r' + ' '.repeat(process.stdout.columns || 80) + '\r',
    );
    process.stdout.write(line);
  }

  /**
   * completion progress display
   */
  complete(): void {
    process.stdout.write('\n');
  }
}
