import PosWorkspace from '../pos-workspace';

export default function OverviewPage({
  children: _children,
  sales: _sales,
  pie_stats: _pieStats,
  bar_stats: _barStats,
  area_stats: _areaStats
}: {
  children: React.ReactNode;
  sales: React.ReactNode;
  pie_stats: React.ReactNode;
  bar_stats: React.ReactNode;
  area_stats: React.ReactNode;
}) {
  return <PosWorkspace />;
}
