import React, { useState, useCallback, useRef } from 'react';
import { Check, CircleAlert, X } from 'lucide-react';

import { ToastContext, type ToastType } from './toast-context';

interface Toast {
  id: number;
  message: string;
  type: ToastType;
}

export function ToastProvider({ children }: { children: React.ReactNode }) {
  const [toasts, setToasts] = useState<Toast[]>([]);

  const nextId = useRef(0);

  const showToast = useCallback((message: string, type: ToastType) => {
    const id = ++nextId.current;
    setToasts(prev => [...prev, { id, message, type }]);
    setTimeout(() => {
      setToasts(prev => prev.filter(t => t.id !== id));
    }, 4000);
  }, []);

  return (
    <ToastContext.Provider value={{ showToast }}>
      {children}
      <div className="toast-container">
        {toasts.map(toast => (
          <div key={toast.id} className={`toast toast-${toast.type}`} role="status">
            {toast.type === 'success' ? <Check size={16} /> : <CircleAlert size={16} />}
            <span>{toast.message}</span>
            <button className="icon-button" aria-label="Dismiss notification" onClick={() => setToasts(prev => prev.filter(t => t.id !== toast.id))}>
              <X size={15} />
            </button>
          </div>
        ))}
      </div>
    </ToastContext.Provider>
  );
}
