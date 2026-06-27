"use strict";
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
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.GridItem = exports.GridList = void 0;
const react_1 = require("react");
const classnames_1 = __importDefault(require("classnames"));
const index_module_less_1 = __importDefault(require("./index.module.less"));
const GridList = ({ children, averageItemWidth = 300, gap = 16, className, onResize, ...resetProps }) => {
    const gridListRef = (0, react_1.useRef)(null);
    const [repeatCount, setRepeatCount] = (0, react_1.useState)(0);
    (0, react_1.useEffect)(() => {
        if (gridListRef.current) {
            const resizeObserver = new ResizeObserver(entries => {
                for (let entry of entries) {
                    const { width } = entry.contentRect;
                    const itemWidth = averageItemWidth;
                    const newRepeatCount = Math.max(1, ~~(width / (itemWidth + gap)));
                    const renderItemWidth = (width - (newRepeatCount - 1) * gap) / newRepeatCount;
                    setRepeatCount(newRepeatCount);
                    onResize === null || onResize === void 0 ? void 0 : onResize(renderItemWidth, newRepeatCount);
                }
            });
            resizeObserver.observe(gridListRef.current);
        }
    }, [averageItemWidth, gap, onResize]);
    return (<div {...resetProps} ref={gridListRef} className={(0, classnames_1.default)(index_module_less_1.default.gridList, className)} style={{
            gridTemplateColumns: `repeat(${repeatCount}, 1fr)`,
            gridGap: `${gap}px`,
        }}>
      {repeatCount ? children : null}
    </div>);
};
exports.GridList = GridList;
const GridItem = ({ children, disabled = false, className, ...restProps }) => (<div {...restProps} className={(0, classnames_1.default)(index_module_less_1.default.gridItem, className, {
        [index_module_less_1.default.disabled]: disabled,
    })}>
    {children}
  </div>);
exports.GridItem = GridItem;
//# sourceMappingURL=gridList.jsx.map