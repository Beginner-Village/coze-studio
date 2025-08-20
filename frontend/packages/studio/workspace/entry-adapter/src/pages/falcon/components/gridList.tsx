import { useRef, useEffect, useState } from 'react';

import cls from 'classnames';

import styles from './index.module.less';

export const GridList = ({
  children,
  averageItemWidth = 300,
  gap = 16,
  className,
  onResize,
  ...resetProps
}) => {
  const gridListRef = useRef<HTMLDivElement>(null);
  const [repeatCount, setRepeatCount] = useState(0);

  useEffect(() => {
    if (gridListRef.current) {
      const resizeObserver = new ResizeObserver(entries => {
        for (let entry of entries) {
          const { width } = entry.contentRect;
          const itemWidth = averageItemWidth;
          const newRepeatCount = Math.max(1, ~~(width / (itemWidth + gap)));
          const renderItemWidth =
            (width - (newRepeatCount - 1) * gap) / newRepeatCount;
          setRepeatCount(newRepeatCount);
          onResize?.(renderItemWidth, newRepeatCount);
        }
      });
      resizeObserver.observe(gridListRef.current);
    }
  }, [averageItemWidth, gap, onResize]);

  return (
    <div
      {...resetProps}
      ref={gridListRef}
      className={cls(styles.gridList, className)}
      style={{
        gridTemplateColumns: `repeat(${repeatCount}, 1fr)`,
        gridGap: `${gap}px`,
      }}
    >
      {repeatCount ? children : null}
    </div>
  );
};

export const GridItem = ({
  children,
  disabled = false,
  className,
  ...restProps
}) => (
  <div
    {...restProps}
    className={cls(styles.gridItem, className, {
      [styles.disabled]: disabled,
    })}
  >
    {children}
  </div>
);
