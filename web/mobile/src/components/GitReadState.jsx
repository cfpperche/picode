export default function GitReadState({ error, loading, onRetry, children }) {
  if (error) return <div className="m-git-state" role="alert"><p>{error}</p>{onRetry ? <button type="button" className="btn" onClick={onRetry}>Retry</button> : null}</div>;
  if (loading) return <div className="m-git-skeleton" role="status" aria-label="Loading Git" aria-busy="true">{[0, 1, 2, 3, 4].map(i => <span key={i} />)}</div>;
  return children || null;
}
