using System;
using System.Linq;
using ExampleORM.Extensions;
using Microsoft.EntityFrameworkCore;

namespace ExampleORM.Models
{
    public interface IUpdateable
    {
        DateTime UpdatedAt { get; set; }
    }

    public class ExampleContext : DbContext
    {
        public DbSet<User> Users { get; set; }

        protected override void OnConfiguring(DbContextOptionsBuilder optionsBuilder)
        {
            optionsBuilder.UseNpgsql("Host=localhost;Port=9000;Username=pguser;Password=;Database=testdb");
        }

        private void EnsureCamelCase(ModelBuilder builder)
        {
            foreach (var entity in builder.Model.GetEntityTypes())
            {
                // Replace table names
                entity.Relational().TableName = entity.Relational().TableName.ToSnakeCase();

                // Replace column names            
                foreach (var property in entity.GetProperties())
                {
                    property.Relational().ColumnName = property.Name.ToSnakeCase();
                }

                foreach (var key in entity.GetKeys())
                {
                    key.Relational().Name = key.Relational().Name.ToSnakeCase();
                }

                foreach (var key in entity.GetForeignKeys())
                {
                    key.Relational().Name = key.Relational().Name.ToSnakeCase();
                }

                foreach (var index in entity.GetIndexes())
                {
                    index.Relational().Name = index.Relational().Name.ToSnakeCase();
                }
            }
        }

        protected override void OnModelCreating(ModelBuilder modelBuilder)
        {
            base.OnModelCreating(modelBuilder);

            EnsureCamelCase(modelBuilder);
            var b = modelBuilder.Entity<User>();
            b.Property(u => u.Name).IsRequired().HasDefaultValue("Player");
            b.Property(u => u.UpdatedAt).HasDefaultValueSql("NOW()");
            b.Property(u => u.Id);
            b.HasKey(u => u.Id);
            modelBuilder.HasPostgresExtension("uuid-ossp")
                .Entity<User>()
                .Property(u => u.Token)
                .HasDefaultValueSql("uuid_generate_v4()");
        }

        public override int SaveChanges()
        {
            var currentDateTime = DateTime.Now;
            var entries = ChangeTracker.Entries().ToList();

            var updatedEntries = entries.Where(e => e.Entity is IUpdateable)
                .Where(e => e.State == EntityState.Modified)
                .ToList();

            updatedEntries.ForEach(e =>
            {
                ((IUpdateable) e.Entity).UpdatedAt = currentDateTime;
                Console.WriteLine(((IUpdateable) e.Entity).UpdatedAt);
            });

            return base.SaveChanges();
        }
    }

    public class User : IUpdateable
    {
        public Guid Id { get; set; }
        public string Name { get; set; }
        public Guid Token { get; set; }
        public DateTime UpdatedAt { get; set; }
    }
}