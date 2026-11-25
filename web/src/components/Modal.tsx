import { useEffect, useId, useRef, type ReactNode } from 'react';
import { X } from 'lucide-react';

interface ModalProps {
  isOpen: boolean;
  onClose: () => void;
  title: string;
  children: ReactNode;
  busy?: boolean;
}

export function Modal({ isOpen, onClose, title, children, busy }: ModalProps) {
  const dialog = useRef<HTMLDialogElement>(null);
  const titleId = useId();

  useEffect(() => {
    const element = dialog.current;
    if (isOpen && element && !element.open) element.showModal();
    if (!isOpen && element?.open) element.close();
  }, [isOpen]);

  return (
    <dialog
      ref={dialog}
      className="modal"
      aria-labelledby={titleId}
      onCancel={event => {
        event.preventDefault();
        if (!busy) onClose();
      }}
    >
      {isOpen && (
        <>
          <div className="modal-header">
            <h2 id={titleId}>{title}</h2>
            <button type="button" onClick={onClose} className="icon-button" aria-label="Close dialog" disabled={busy}>
              <X size={18} />
            </button>
          </div>
          {children}
        </>
      )}
    </dialog>
  );
}
