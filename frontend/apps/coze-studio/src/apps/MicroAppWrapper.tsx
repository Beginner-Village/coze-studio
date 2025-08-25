import { useEffect, useRef } from 'react';
import { useParams } from 'react-router-dom';
import { loadMicroApp } from 'qiankun';
import { microApps } from './config'
import './microapp.scss'

interface MicroAppWrapperProps {
  appName: string;
}

export const MicroAppWrapper = ({ appName }: MicroAppWrapperProps) => {
  const { space_id } = useParams();
  const containerRef = useRef<HTMLDivElement>(null);
  const microAppRef = useRef<any>(null);
  useEffect(() => {
    if (containerRef.current) {
      let appInfo = microApps.find((app) => app.name === appName);
      if(!appInfo) return
      let appPath = window.localStorage.getItem(`subapp_${appInfo.name}`);

      // 根据微应用名称加载对应的微应用
      const microApp = loadMicroApp({
        name: appName,
        entry: appPath || appInfo.entry,
        container: containerRef.current,
        props: { spaceId: space_id },
      });
      microAppRef.current = microApp;
      return () => {
        // 卸载微应用
        microApp.mountPromise.then(() => {
          microApp.unmount();
        });
      };
    }
  }, [appName]);
  return (
    <div className="h-full w-full">
      {/* 微应用加载时的加载指示器 */}
      {/* 可以根据需要添加 loading 效果 */}
      <div ref={containerRef} className="subapp-view" style={{ height: '100%' }}></div>
    </div>
  );
}
