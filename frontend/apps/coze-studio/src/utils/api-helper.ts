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

/**
 * API响应统一处理工具
 *
 * 解决问题：API客户端有时会将成功响应（code=200）当作错误抛出
 * 使用方法：
 *
 * import { callApi, isSuccessResponse } from '@/utils/api-helper';
 *
 * // 方式1：使用callApi包装
 * const result = await callApi(() => space_management.GetSpaceList({ page: 1 }));
 * if (result.success) {
 *   setSpaceList(result.data);
 * } else {
 *   showError(result.error);
 * }
 *
 * // 方式2：在catch中使用isSuccessResponse检查
 * try {
 *   const response = await api.someMethod();
 * } catch (error) {
 *   const successData = extractSuccessData(error);
 *   if (successData) {
 *     // 处理数据
 *   }
 * }
 */

// 标准响应码
const SUCCESS_CODES = [0, 200, '0', '200'];

/**
 * 检查响应码是否表示成功
 */
export function isSuccessCode(code: unknown): boolean {
  return SUCCESS_CODES.includes(code as string | number);
}

/**
 * 检查错误对象是否实际上是成功响应
 */
export function isSuccessResponse(error: unknown): boolean {
  if (!error || typeof error !== 'object') return false;

  const err = error as Record<string, unknown>;
  return isSuccessCode(err.code);
}

/**
 * 从错误对象中提取成功响应的数据
 */
export function extractSuccessData<T>(error: unknown): T | null {
  if (!isSuccessResponse(error)) return null;

  const err = error as Record<string, unknown>;

  // 尝试多种可能的数据路径
  const responseData = (err.response as Record<string, unknown>)?.data || err.data || err;

  if (responseData && typeof responseData === 'object') {
    const data = responseData as Record<string, unknown>;
    if (data.data !== undefined) {
      return data.data as T;
    }
    return responseData as T;
  }

  return null;
}

/**
 * API调用结果类型
 */
export interface ApiResult<T> {
  success: boolean;
  data: T | null;
  error: string | null;
  code: number | string | null;
}

/**
 * 统一的API调用包装函数
 * 自动处理成功响应被当作错误的情况
 *
 * @param apiCall - API调用函数
 * @returns 标准化的API结果
 *
 * @example
 * const result = await callApi(() => userApi.getUser(userId));
 * if (result.success) {
 *   console.log('User:', result.data);
 * } else {
 *   console.error('Error:', result.error);
 * }
 */
export async function callApi<T>(
  apiCall: () => Promise<{ code?: number | string; data?: T; msg?: string }>
): Promise<ApiResult<T>> {
  try {
    const response = await apiCall();

    if (isSuccessCode(response.code)) {
      return {
        success: true,
        data: response.data ?? null,
        error: null,
        code: response.code ?? null,
      };
    }

    return {
      success: false,
      data: null,
      error: response.msg || '请求失败',
      code: response.code ?? null,
    };
  } catch (error: unknown) {
    // 检查是否是被错误处理的成功响应
    if (isSuccessResponse(error)) {
      const data = extractSuccessData<T>(error);
      return {
        success: true,
        data,
        error: null,
        code: (error as Record<string, unknown>).code as number | string,
      };
    }

    // 真正的错误
    const err = error as Record<string, unknown>;
    return {
      success: false,
      data: null,
      error: (err.message as string) || (err.msg as string) || '请求失败',
      code: (err.code as number | string) ?? null,
    };
  }
}

/**
 * 批量API调用，并行执行多个API请求
 */
export async function callApisParallel<T extends readonly unknown[]>(
  apiCalls: { [K in keyof T]: () => Promise<{ code?: number | string; data?: T[K]; msg?: string }> }
): Promise<{ [K in keyof T]: ApiResult<T[K]> }> {
  const results = await Promise.all(apiCalls.map(call => callApi(call)));
  return results as { [K in keyof T]: ApiResult<T[K]> };
}

/**
 * 带重试的API调用
 */
export async function callApiWithRetry<T>(
  apiCall: () => Promise<{ code?: number | string; data?: T; msg?: string }>,
  options: {
    maxRetries?: number;
    retryDelay?: number;
    shouldRetry?: (result: ApiResult<T>) => boolean;
  } = {}
): Promise<ApiResult<T>> {
  const { maxRetries = 3, retryDelay = 1000, shouldRetry } = options;

  let lastResult: ApiResult<T> | null = null;

  for (let attempt = 0; attempt <= maxRetries; attempt++) {
    lastResult = await callApi(apiCall);

    if (lastResult.success) {
      return lastResult;
    }

    // 检查是否应该重试
    if (shouldRetry && !shouldRetry(lastResult)) {
      return lastResult;
    }

    // 最后一次尝试不需要等待
    if (attempt < maxRetries) {
      await new Promise(resolve => setTimeout(resolve, retryDelay));
    }
  }

  return lastResult!;
}
