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
 * Dependency treeShaking Removes Extraneous Configuration (Argus)
 */
const sdkRegion = 'cn';

declare const STUDIO_HOST: string | undefined;

const studioHost: string = (() => {
  // Build-time injection takes priority; fall back to location.origin (dev mode)
  if (typeof STUDIO_HOST !== 'undefined' && STUDIO_HOST) {
    return STUDIO_HOST;
  }
  if (typeof location !== 'undefined') {
    return location.origin;
  }
  return '';
})();

export const iframeAppHost = studioHost;
export const cozeOfficialHost = studioHost;

export const openApiCdnUrlByRegion = IS_OVERSEA
  ? // cp-disable-next-line
    'https://sf16-sg.tiktokcdn.com/obj/eden-sg/rkzild_lgvj/ljhwZthlaukjlkulzlp/'
  : // cp-disable-next-line
    'https://lf3-static.bytednsdoc.com/obj/eden-cn/rkzild_lgvj/ljhwZthlaukjlkulzlp/';

// The user needs to modify the baseurl here to open the domain name configuration of the API interface
export const openApiHostByRegion =
  typeof location !== 'undefined' ? location.origin : 'https://api.xxx.com';
export const openApiHostByRegionWithToken = openApiHostByRegion;

export const openSdkPrefix = '/static/sdk';
export const getOpenSDKUrl = (_version: string) =>
  `${openApiHostByRegion}${openSdkPrefix}/ynet-web-sdk.js`;

export const getOpenSDKPath = (_version: string) =>
  `${openSdkPrefix}/ynet-web-sdk.js`;

export const eventMeta = {
  region: sdkRegion,
  is_release: false,
  dev: false,
};
