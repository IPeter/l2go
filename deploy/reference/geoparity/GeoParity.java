import java.io.BufferedReader;
import java.io.IOException;
import java.io.PrintWriter;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.List;

import org.l2jmobius.gameserver.config.GeoEngineConfig;
import org.l2jmobius.gameserver.entity.Location;
import org.l2jmobius.gameserver.geoengine.GeoEngine;
import org.l2jmobius.gameserver.geoengine.pathfinding.GeoLocation;
import org.l2jmobius.gameserver.geoengine.pathfinding.PathFinding;

/**
 * Answers geodata queries with Mobius's GeoEngine and PathFinding, for the L2Go geoparity test.
 * <p>
 * Usage: java -cp 'mobius/libs/*:classes' GeoParity GEODATA_DIR QUERIES ANSWERS
 * <p>
 * Run it in an empty working folder: Mobius then finds no Doors.xml or FenceData.xml, and every answer is terrain only. The settings are GeoEngine.ini's defaults. Each query line is a kind and integers; each answer line matches its query:
 *
 * <pre>
 * h x y z                      getHeight                      height
 * s x y z                      getSpawnHeight                 height
 * b x y z                      isCompletelyBlocked(geoX, geoY, z)   true|false
 * see x y z tx ty tz           canSeeTarget (terrain only)    true|false
 * move x y z tx ty tz          canMoveToTarget                true|false
 * valid x y z tx ty tz         getValidLocation               x y z
 * path x y z tx ty tz p        PathFinding.findPath, p 1 for a playable   null | n x1 y1 z1 ...
 * </pre>
 */
public class GeoParity
{
	public static void main(String[] args) throws IOException
	{
		if (args.length != 3)
		{
			System.err.println("usage: GeoParity GEODATA_DIR QUERIES ANSWERS");
			System.exit(2);
		}

		// GeoEngine.ini, as shipped.
		GeoEngineConfig.GEODATA_PATH = Path.of(args[0]);
		GeoEngineConfig.PATHFINDING = 2;
		GeoEngineConfig.PATHFIND_BUFFERS = "100x6;128x6;192x6;256x4;320x4;384x4;500x2";
		GeoEngineConfig.LOW_WEIGHT = 0.5f;
		GeoEngineConfig.MEDIUM_WEIGHT = 2;
		GeoEngineConfig.HIGH_WEIGHT = 3;
		GeoEngineConfig.ADVANCED_DIAGONAL_STRATEGY = false;
		GeoEngineConfig.AVOID_OBSTRUCTED_PATH_NODES = true;
		GeoEngineConfig.DIAGONAL_WEIGHT = 0.707f;
		GeoEngineConfig.MAX_POSTFILTER_PASSES = 3;

		final GeoEngine geo = GeoEngine.getInstance();
		final PathFinding pathFinding = PathFinding.getInstance();
		try (BufferedReader in = Files.newBufferedReader(Path.of(args[1]));
			PrintWriter out = new PrintWriter(Files.newBufferedWriter(Path.of(args[2]))))
		{
			String line;
			while ((line = in.readLine()) != null)
			{
				line = line.trim();
				if (line.isEmpty())
				{
					continue;
				}

				final String[] fields = line.split(" +");
				final int[] v = new int[fields.length - 1];
				for (int i = 1; i < fields.length; i++)
				{
					v[i - 1] = Integer.parseInt(fields[i]);
				}

				switch (fields[0])
				{
					case "h" -> out.println(geo.getHeight(v[0], v[1], v[2]));
					case "s" -> out.println(geo.getSpawnHeight(v[0], v[1], v[2]));
					case "b" -> out.println(geo.isCompletelyBlocked(GeoEngine.getGeoX(v[0]), GeoEngine.getGeoY(v[1]), v[2]));
					case "see" -> out.println(geo.canSeeTarget(v[0], v[1], v[2], v[3], v[4], v[5]));
					case "move" -> out.println(geo.canMoveToTarget(v[0], v[1], v[2], v[3], v[4], v[5], 0));
					case "valid" ->
					{
						final Location location = geo.getValidLocation(v[0], v[1], v[2], v[3], v[4], v[5], 0);
						out.println(location.getX() + " " + location.getY() + " " + location.getZ());
					}
					case "path" ->
					{
						final List<GeoLocation> path = pathFinding.findPath(v[0], v[1], v[2], v[3], v[4], v[5], 0, v[6] != 0);
						if (path == null)
						{
							out.println("null");
						}
						else
						{
							final StringBuilder sb = new StringBuilder().append(path.size());
							for (GeoLocation node : path)
							{
								sb.append(' ').append(node.getX()).append(' ').append(node.getY()).append(' ').append(node.getZ());
							}
							out.println(sb);
						}
					}
					default -> throw new IllegalArgumentException("unknown query: " + line);
				}
			}
		}

		// Mobius's singletons may have started threads.
		System.exit(0);
	}
}
