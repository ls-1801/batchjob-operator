package de.tuberlin.batchjoboperator.spark.webcrawler

import com.squareup.okhttp._
import org.apache.spark.sql.{Dataset, SparkSession}
import org.apache.spark.util.LongAccumulator
import org.codehaus.jackson.map.ObjectMapper
import org.slf4j.LoggerFactory

import java.io.IOException
import java.net.URLEncoder
import java.util.concurrent.{ConcurrentLinkedQueue, CountDownLatch}
import scala.annotation.tailrec
import scala.jdk.CollectionConverters.collectionAsScalaIterableConverter

object WebCrawlerApplication {
  val client = new OkHttpClient();
  val mapper = new ObjectMapper()
  var log = LoggerFactory.getLogger(WebCrawlerApplication.getClass)
  val BATCH_SIZE = 50;

  def getLinksOnPages(longAccumulator: LongAccumulator, pages: List[String]): List[String] = {
    val latch = new CountDownLatch(pages.size)
    val results = new ConcurrentLinkedQueue[List[String]]()
    val failures = new ConcurrentLinkedQueue[String]()
    val start = System.nanoTime();
    pages.foreach { page =>
      val request = new Request.Builder()
        .url("https://en.wikipedia.org/w/api" +
          ".php?action=query&format=json&prop=links&titles=" + URLEncoder.encode(page))
        .get().build()

      client.newCall(request).enqueue(new Callback {
        override def onFailure(request: Request, e: IOException): Unit = {
          latch.countDown()
          failures.add(page)
        }

        override def onResponse(response: Response): Unit = {
          try {
            val json = mapper.readTree(response.body().byteStream());
            val query = json.get("query").get("pages");
            val articleId = query.getFieldNames.next()
            val linksNode = query.get(articleId).get("links");

            if (linksNode == null)
              return

            val links = 0 until linksNode.size() map { i =>
              linksNode.get(i).get("title").asText()
            }

            results.add(links.toList)
          } finally {
            latch.countDown()
          }
        }
      })
    }
    latch.await()
    longAccumulator.add(pages.size);
    val end = System.nanoTime();
    log.info("{} Requests took {}ms", pages.size, (end - start) / (1000 * 1000))

    if (failures.size() > 0) {
      log.warn("Failures: {}", failures.size())
    }

    results.asScala.flatMap(_.toStream).toList
  }

  def main(args: Array[String]): Unit = {
    val start = args(0)
    val limit = args(1).toInt
    val target = args(2)

    val spark = SparkSession
      .builder
      //      .master("local[5]")
      .appName("Spark Web Crawler")
      .getOrCreate()

    import spark.implicits._
    val accumulator = spark.sparkContext.longAccumulator

    val links = spark.createDataset(getLinksOnPages(accumulator, List(start)))
    val allLinks = spark.createDataset(List(start))

    val requestsDone = recursive(limit, accumulator, target, spark, links, allLinks)
    log.info("Done: {}", requestsDone)
    spark.stop()
  }

  @tailrec
  def recursive(limit: Long, accumulator: LongAccumulator, target: String, spark: SparkSession, links: Dataset[String],
                allLinks: Dataset[String])
  : Long = {
    import spark.implicits._

    // Keep Track of all Links
    val allLinks2 = allLinks.unionAll(links)

    val nLinks = links.count()
    // Partition to batch size, but not more than 100
    val nPartitions = math.min((nLinks / BATCH_SIZE) + 1, 100).intValue()
    log.info("Partition {} links across {} partitions", nLinks, nPartitions)

    val coalescedAndGrouped = links
      .repartition(nPartitions)
      .mapPartitions(_.grouped(BATCH_SIZE+10))

    log.info("Partitions: {}", coalescedAndGrouped.rdd.getNumPartitions)

    val newLinksThisIteration = coalescedAndGrouped.flatMap { links => getLinksOnPages(accumulator, links.toList) }
      // Ignore duplicates
      .distinct()
      // Do not check the same link twice
      .except(allLinks2)
      .cache()


    // Target was found
    if (!newLinksThisIteration.filter(s => target.equals(s)).isEmpty) {
      log.info("Target: '{}' was found", target)
      return accumulator.value
    }

    if ( // No new links were found
        newLinksThisIteration.count() == 0 ||
        // Limit was reached
        accumulator.value >= limit) {
      return accumulator.value
    }



    // Limit links for next iteration if necessary
    if (newLinksThisIteration.count() > limit - accumulator.value) {
      val limitedLinks = newLinksThisIteration.limit(math.max(limit - accumulator.value, 0).intValue()).cache()
      return recursive(limit, accumulator, target, spark, limitedLinks, allLinks2)
    } else {
      return recursive(limit, accumulator, target, spark, newLinksThisIteration, allLinks2)
    }

  }
}