/**
 * Zone Information API Route
 *
 * Returns zone configuration and detection information.
 * Protected by withZone middleware.
 *
 * GET /api/zones - Get all zones
 * GET /api/zones?path=/some/path - Detect zone for a path
 */

import { NextRequest, NextResponse } from 'next/server';
import { withZone, CodexRouteServer, CodexRouteAnalyzer, ZONE_NAMES } from '@/lib/zones';

const analyzer = new CodexRouteAnalyzer();

export const GET = withZone(async (request: NextRequest) => {
  const searchParams = request.nextUrl.searchParams;
  const pathToAnalyze = searchParams.get('path');

  // If path provided, detect zone for it
  if (pathToAnalyze) {
    const detection = analyzer.getZoneForAppRoute(pathToAnalyze);
    const zone = CodexRouteServer.getZone(detection.zone_id);

    return NextResponse.json({
      success: true,
      path: pathToAnalyze,
      detection: {
        zone_id: detection.zone_id,
        zone_name: detection.zone_name,
        confidence: detection.confidence,
        matched_keywords: detection.matched_keywords,
      },
      zone: zone
        ? {
            name: zone.name,
            description: zone.description,
            rules: zone.rules,
          }
        : null,
    });
  }

  // Return all zones
  const allZones = CodexRouteServer.All();
  const zonesArray = Object.entries(allZones).map(([id, zone]) => ({
    id: parseInt(id),
    ...zone,
  }));

  return NextResponse.json({
    success: true,
    zones: zonesArray,
    zone_names: ZONE_NAMES,
    strategies: analyzer.getAllStrategies(),
  });
});

export const runtime = 'nodejs';
