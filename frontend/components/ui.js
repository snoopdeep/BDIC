export function PageHeader({ title, children }) {
  return (
    <div className="mb-6">
      <h1 className="text-3xl font-bold">{title}</h1>
      {children}
    </div>
  );
}

export function Panel({ children, title }) {
  return (
    <div className="border rounded-lg p-6 bg-white shadow-sm">
      {title && <h2 className="text-xl font-bold mb-4">{title}</h2>}
      {children}
    </div>
  );
}

export function DataTable({ columns, data, loading }) {
  if (loading) return <div className="p-4">Loading...</div>;
  if (!data || data.length === 0) return <div className="p-4">No data</div>;

  return (
    <table className="w-full">
      <thead>
        <tr>
          {columns && columns.map((col, i) => <th key={i} className="text-left p-2">{col.label}</th>)}
        </tr>
      </thead>
      <tbody>
        {data.map((row, i) => (
          <tr key={i} className="border-t">
            {columns && columns.map((col, j) => (
              <td key={j} className="p-2">{row[col.key]}</td>
            ))}
          </tr>
        ))}
      </tbody>
    </table>
  );
}

export function EmptyState({ title, message }) {
  return (
    <div className="p-8 text-center">
      <p className="text-xl font-bold">{title}</p>
      <p className="text-gray-600">{message}</p>
    </div>
  );
}

export function LoadingBlock() {
  return <div className="p-4 animate-pulse">Loading...</div>;
}

export function Alert({ type, children }) {
  const bgColor = type === 'error' ? 'bg-red-100' : 'bg-blue-100';
  return <div className={`${bgColor} p-4 rounded mb-4`}>{children}</div>;
}

export function Badge({ type, children }) {
  const colors = {
    success: 'bg-green-100 text-green-800',
    warning: 'bg-yellow-100 text-yellow-800',
    error: 'bg-red-100 text-red-800',
    default: 'bg-gray-100 text-gray-800'
  };
  return <span className={`px-2 py-1 rounded text-sm ${colors[type] || colors.default}`}>{children}</span>;
}
